package ginhttp

import (
	"context"
	"github.com/liuliliujian/go-infra-com/config"
	"github.com/liuliliujian/go-infra-com/util/ginutil"
	"errors"
	"fmt"
	"github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	errs "github.com/pkg/errors"
	"github.com/thoas/go-funk"
	"go.uber.org/zap"
	"net"
	"net/http"
	"strings"
	"time"
)

type Options struct {
	Port int
	Mode string
}

var (
	supportedModes = []string{gin.DebugMode, gin.TestMode, gin.ReleaseMode}
)

func NewOptions(c config.Config) (*Options, error) {
	o := &Options{
		Port: 9090,
		Mode: gin.DebugMode,
	}
	if err := c.UnmarshalKey("gin", o); err != nil {
		return nil, errs.WithMessage(err, "invalid gin config options")
	}
	if o.Port <= 0 || o.Port >= 60000 {
		return nil, errors.New("gin port must between (0, 60000)")
	}
	o.Mode = strings.ToLower(o.Mode)
	if !funk.ContainsString(supportedModes, o.Mode) {
		return nil, errors.New(fmt.Sprintf("invalid gin mode[%v], only support:%v", o.Mode, supportedModes))
	}
	return o, nil
}

type RouterConfigurer func(router *gin.Engine)

type GinModuleConfigurer struct {
	OverrideDefaultMiddlewares bool
	RouterConfigurer           RouterConfigurer
}

func NewRouter(o *Options, logger *zap.Logger, configurer GinModuleConfigurer) (*gin.Engine, error) {
	gin.SetMode(o.Mode)
	router := gin.New()
	router.Use(gin.Recovery()) //确保ginzap模块有问题时的保障, 观察一段时间，如果没问题可以移除
	router.Use(ginzap.Ginzap(logger, "2006-01-02 15:04:05", false))
	router.Use(ginzap.RecoveryWithZap(logger, true))

	if !configurer.OverrideDefaultMiddlewares {
		//todo add built-in middlewares, for example auth, monitor
	}

	router.NoRoute(func(c *gin.Context) {
		ginutil.ApiError(c, ginutil.Status_Api_NotFound, "api not found")
	})
	router.NoMethod(func(c *gin.Context) {
		ginutil.ApiError(c, ginutil.Status_Method_NotSupport, "method not support")
	})

	//for health check
	router.GET("/health", func(context *gin.Context) {
		context.String(http.StatusOK, "OK")
	})

	if configurer.RouterConfigurer != nil {
		configurer.RouterConfigurer(router)
	}

	return router, nil
}

type Server struct {
	options    *Options
	logger     *zap.Logger
	router     *gin.Engine
	httpServer *http.Server
}

func NewServer(o *Options, logger *zap.Logger, router *gin.Engine) (*Server, error) {
	return &Server{
		options: o,
		logger:  logger,
		router:  router,
	}, nil
}

func (s *Server) Start() error {
	s.httpServer = &http.Server{
		Addr:           fmt.Sprintf(":%d", s.options.Port),
		Handler:        s.router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	failChan := make(chan error, 1)
	s.logger.Info("starting gin http server...")
	listener, err := net.Listen("tcp", s.httpServer.Addr)
	if err != nil {
		s.logger.Error(fmt.Sprintf("failed to start gin http server[%d]", s.options.Port), zap.Error(err))
		return err
	}
	go func() {
		if err := s.httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			defer func() { failChan <- err }()
			s.logger.Error("failed to start gin http server", zap.Error(err))
		}
	}()
	select {
	case err := <-failChan:
		return err
	case <-time.After(1 * time.Second):
		s.logger.Sugar().Infof("succeed to start gin http server[%d]", s.options.Port)
	}
	return nil
}

func (s *Server) Stop() error {
	s.logger.Info("stopping gin http server...")
	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error("failed to stop gin http server")
		return err
	}
	s.logger.Info("succeed to stop gin http server")
	return nil
}

//without module configurer
var ProviderSet = wire.NewSet(NewRouter, NewServer, NewOptions, wire.Value(GinModuleConfigurer{}))

//need module configurer
var ProviderSetWithConfigurer = wire.NewSet(NewRouter, NewServer, NewOptions)
