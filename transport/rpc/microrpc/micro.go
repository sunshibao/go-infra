package microrpc

import (
	"context"
	"github.com/liuliliujian/go-infra-com/config"
	"github.com/liuliliujian/go-infra-com/transport/rpc/microrpc/middleware/logerr"
	"github.com/liuliliujian/go-infra-com/util/syncutil"
	"errors"
	"fmt"
	"github.com/google/wire"
	"github.com/micro/cli"
	"github.com/micro/go-micro"
	"github.com/micro/go-micro/client"
	merrs "github.com/micro/go-micro/errors"
	"github.com/micro/go-micro/metadata"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/registry/etcd"
	"github.com/micro/go-micro/server"
	"github.com/micro/go-micro/transport"
	"github.com/micro/go-micro/util/log"
	"github.com/micro/go-plugins/wrapper/select/roundrobin"
	errs "github.com/pkg/errors"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"strings"
	"sync"
	"time"
)

const (
	Registry_Prefix_Etcd = "etcd://"
)

type Options struct {
	Name     string
	Registry string
	Metadata map[string]string
	Server   *ServerOptions
	Client   *ClientOptions
	//下面字段放这很尴尬, 先这样吧
	StartChan chan error
	StopFunc  context.CancelFunc
	StopChan  chan error
}

type ServerOptions struct {
	RegisterTTL      time.Duration
	RegisterInterval time.Duration
	LogError         bool
}

type ClientOptions struct {
	RequestTimeout time.Duration
	DialTimeout    time.Duration
	Retries        int
	Loadbalance    string
	LogError       bool
}

func NewOptions(c config.Config, logger *zap.Logger) (*Options, error) {
	o := &Options{
		Name: config.GetApplicationName(c),
	}

	if err := c.UnmarshalKey("micro", o); err != nil {
		return nil, errs.WithMessage(err, "invalid micro service config options")
	}

	if o.Name == "" {
		return nil, errors.New("micro service's name is required")
	}

	if o.Registry == "" {
		return nil, errors.New("micro service's registry is required")
	}

	o.StartChan = make(chan error, 1)
	o.StopChan = make(chan error, 1)
	logger.Sugar().Infof("build micro service from registry[%s]", o.Registry)
	return o, nil
}

type MicroModuleConfigurer struct {
	ClientWrappersExtender  ExtendClientWrappers
	CallWrappersExtender    ExtendCallWrappers
	HandlerWrappersExtender ExtendHandlerWrappers
	ServiceHandlerRegister  ServiceHandlerRegister
}

type ExtendHandlerWrappers func([]server.HandlerWrapper) []server.HandlerWrapper
type ExtendClientWrappers func([]client.Wrapper) []client.Wrapper
type ExtendCallWrappers func([]client.CallWrapper) []client.CallWrapper

func NewService(o *Options, logger *zap.Logger, configurer MicroModuleConfigurer) (micro.Service, error) {
	log.SetLogger(microzap{logger})

	options := make([]micro.Option, 0, 10)
	options = append(options, micro.Name(o.Name))

	//start client options, must before some micro options, e.g. micro.registry
	clientOptions := make([]client.Option, 0, 10)
	requestTimeout := 10 * time.Second
	if o.Client != nil && o.Client.RequestTimeout > 0 {
		requestTimeout = o.Client.RequestTimeout
	}
	clientOptions = append(clientOptions, client.RequestTimeout(requestTimeout))

	dialTimeout := 10 * time.Second
	if o.Client != nil && o.Client.DialTimeout > 0 {
		dialTimeout = o.Client.DialTimeout
	}
	clientOptions = append(clientOptions, client.DialTimeout(dialTimeout))

	clientOptions = append(clientOptions, client.Retry(retryOnConnError(logger)))
	retries := 3
	if o.Client != nil && o.Client.Retries > 0 && o.Client.Retries < 8 {
		retries = o.Client.Retries
	}
	clientOptions = append(clientOptions, client.Retries(retries))

	callWrappers := make([]client.CallWrapper, 0, 10)
	callWrappers = append(callWrappers, recordRoutedNode())
	if configurer.CallWrappersExtender != nil {
		callWrappers = configurer.CallWrappersExtender(callWrappers)
	}
	clientOptions = append(clientOptions, client.WrapCall(callWrappers...))

	options = append(options, micro.Client(client.NewClient(clientOptions...)))

	clientWrappers := make([]client.Wrapper, 0, 10)
	if o.Client != nil && o.Client.Loadbalance == "roundrobin" {
		clientWrappers = append(clientWrappers, roundrobin.NewClientWrapper())
	}
	if o.Client != nil && o.Client.LogError {
		clientWrappers = append(clientWrappers, logerr.NewClientWrapper(logger))
	}
	if configurer.ClientWrappersExtender != nil {
		clientWrappers = configurer.ClientWrappersExtender(clientWrappers)
	}
	options = append(options, micro.WrapClient(clientWrappers...))
	//end client options

	if strings.HasPrefix(o.Registry, Registry_Prefix_Etcd) {
		options = append(options, micro.Registry(
			etcd.NewRegistry(
				registry.Addrs(
					strings.Split(o.Registry[len(Registry_Prefix_Etcd):], ",")...
				),
			),
		))
	}

	ctx, cancel := context.WithCancel(context.Background())
	o.StopFunc = cancel
	options = append(options, micro.Context(ctx))

	if len(o.Metadata) > 0 {
		options = append(options, micro.Metadata(o.Metadata))
	}

	//start server options
	registerTTL := 30 * time.Second
	if o.Server != nil && o.Server.RegisterTTL > 10*time.Second {
		registerTTL = o.Server.RegisterTTL
	}
	options = append(options, micro.RegisterTTL(registerTTL))

	registerInterval := 15 * time.Second
	if o.Server != nil && o.Server.RegisterInterval > 5*time.Second {
		registerInterval = o.Server.RegisterInterval
	}
	if registerInterval > registerTTL/2 {
		registerInterval = registerTTL / 2
	}
	options = append(options, micro.RegisterInterval(registerInterval))

	waitGroup := new(sync.WaitGroup)
	handlerWrappers := make([]server.HandlerWrapper, 0, 10)
	handlerWrappers = append(handlerWrappers, func(handlerFunc server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			waitGroup.Add(1)
			defer waitGroup.Done()
			return handlerFunc(ctx, req, rsp)
		}
	})
	if o.Server != nil && o.Server.LogError {
		handlerWrappers = append(handlerWrappers, logerr.NewHandlerWrapper(logger))
	}
	if configurer.HandlerWrappersExtender != nil {
		handlerWrappers = configurer.HandlerWrappersExtender(handlerWrappers)
	}
	options = append(options, micro.WrapHandler(handlerWrappers...))
	options = append(options, micro.AfterStart(func() error {
		o.StartChan <- nil
		return nil
	}))
	options = append(options, micro.AfterStop(func() error {
		syncutil.WaitGroupTimeout(waitGroup, 6*time.Second) //graceful shutdown util incoming request handled
		o.StopChan <- nil
		return nil
	}))
	//end server options

	options = append(options, micro.Transport(transport.NewTransport(
		transport.Timeout(20*time.Second),
	)))

	//为使micro cli flag解析能正常通过，需要将所有flag适配到micro cli flag
	flags := make([]cli.Flag, 0, 10)
	pflag.VisitAll(func(flag *pflag.Flag) {
		flags = append(flags, cli.StringFlag{
			Name: flag.Name,
		})
	})
	options = append(options, micro.Flags(flags...))

	service := micro.NewService(options...)
	service.Init()
	return service, nil
}

type Server struct {
	logger  *zap.Logger
	options *Options
	service micro.Service
}

type ServiceHandlerRegister func(server.Server)

func NewServer(o *Options, logger *zap.Logger, s micro.Service, configurer MicroModuleConfigurer) (*Server, error) {
	if configurer.ServiceHandlerRegister != nil {
		configurer.ServiceHandlerRegister(s.Server())
	}
	return &Server{
		logger:  logger,
		options: o,
		service: s,
	}, nil
}

func (s *Server) Start() error {
	s.logger.Info("starting micro service server...")
	failChan := make(chan error, 1)
	go func() {
		if err := s.service.Run(); err != nil {
			defer func() { failChan <- err }()
			s.logger.Error("failed to boot micro service server", zap.Error(err))	//maybe start/stop error
		}
	}()
	waitTime := 1 * time.Minute
	select {
	case <-s.options.StartChan:
		s.logger.Info("succeed to start micro service server")
	case err := <-failChan:
		return err
	case <-time.After(waitTime):
		return errors.New(fmt.Sprintf("timeout(%v) waiting for micro service server to start", waitTime))
	}
	return nil
}

func (s *Server) Stop() error {
	s.logger.Info("stopping micro service server...")
	s.options.StopFunc()
	waitTime := 8 * time.Second
	select {
	case <-s.options.StopChan:
	case <-time.After(waitTime):
		return errors.New(fmt.Sprintf("timeout(%v) waiting for micro service server to stop", waitTime))
	}
	s.logger.Info("succeed to stop micro service server")
	return nil
}

type microzap struct {
	*zap.Logger
}

func (m microzap) Log(v ...interface{}) {
	m.Info(fmt.Sprint(v...))
}

func (m microzap) Logf(f string, v ...interface{}) {
	m.Info(fmt.Sprintf(f, v...))
}

func recordRoutedNode() func(callFunc client.CallFunc) client.CallFunc {
	return func(callFunc client.CallFunc) client.CallFunc {
		return func(ctx context.Context, node *registry.Node, req client.Request, rsp interface{}, opts client.CallOptions) error {
			if mb, ok := metadata.FromContext(ctx); ok && node != nil {
				mb["Call-Remote-Addr"] = node.Address //record routed server node
			}
			return callFunc(ctx, node, req, rsp, opts)
		}
	}
}

func retryOnConnError(logger *zap.Logger) client.RetryFunc {
	return func(ctx context.Context, req client.Request, retryCount int, err error) (bool, error) {
		if err == nil {
			return false, nil
		}

		e := merrs.Parse(err.Error())
		if e == nil {
			return false, nil
		}

		switch e.Code {
		// 在服务故障导致未从注册中心移除，或出现连接异常时进行重试，超时不重试，因为超时重试有重复发送请求的风险，对于写操作是致命的
		case 500:
			if e.Id == "go.micro.client" && strings.Contains(e.Detail, "connection error") {
				logger.Info("decide retry with server connection error")
				return true, nil
			}
			if e.Id == "go.micro.client.transport" && strings.Contains(e.Detail, "unexpected EOF") {
				logger.Info("decide retry with transport unexpected eof") //maybe a micro's bug, now retry
				return true, nil
			}
			return false, nil
		default:
			return false, nil
		}
	}
}

//without module configurer
var ProviderSet = wire.NewSet(NewService, NewServer, NewOptions, wire.Value(MicroModuleConfigurer{}))

//need module configurer
var ProviderSetWithConfigurer = wire.NewSet(NewService, NewServer, NewOptions)
