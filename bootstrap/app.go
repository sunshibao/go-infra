package bootstrap

import (
	"github.com/liuliliujian/go-infra-com/config"
	_ "github.com/liuliliujian/go-infra-com/config"
	"github.com/liuliliujian/go-infra-com/transport/http/ginhttp"
	"github.com/liuliliujian/go-infra-com/transport/rpc/microrpc"
	"github.com/liuliliujian/go-infra-com/util/stackutil"
	"flag"
	errs "github.com/pkg/errors"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"
)

func init() {
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
}

type Application struct {
	logger     *zap.Logger
	HttpServer *ginhttp.Server
	RpcServer  *microrpc.Server
}

func NewApp(c config.Config, logger *zap.Logger, httpServer *ginhttp.Server, rpcServer *microrpc.Server) (*Application, error) {
	if c.GetBool(config.CONF_STACKDUMP) {
		stackutil.SetupStackDumper(stackutil.ZapLogger{logger})
	}
	return &Application{
		logger:     logger,
		HttpServer: httpServer,
		RpcServer:  rpcServer,
	}, nil
}

func RunApp(c config.Config, logger *zap.Logger, httpServer *ginhttp.Server, rpcServer *microrpc.Server) (*Application, error) {
	app, err := NewApp(c, logger, httpServer, rpcServer)
	if err != nil {
		return nil, err
	}
	return app, app.Start()
}

func NewHttpApp(c config.Config, logger *zap.Logger, httpServer *ginhttp.Server) (*Application, error) {
	return NewApp(c, logger, httpServer, nil)
}

func RunHttpApp(c config.Config, logger *zap.Logger, httpServer *ginhttp.Server) (*Application, error) {
	return RunApp(c, logger, httpServer, nil)
}

func NewRpcApp(c config.Config, logger *zap.Logger, rpcServer *microrpc.Server) (*Application, error) {
	return NewApp(c, logger, nil, rpcServer)
}

func RunRpcApp(c config.Config, logger *zap.Logger, rpcServer *microrpc.Server) (*Application, error) {
	return RunApp(c, logger, nil, rpcServer)
}

func (a *Application) Start() error {
	if a.HttpServer != nil {
		if err := a.HttpServer.Start(); err != nil {
			return errs.WithMessage(err, "failed to start http server")
		}
	}

	if a.RpcServer != nil {
		if err := a.RpcServer.Start(); err != nil {
			return errs.WithMessage(err, "failed to start rpc server")
		}
	}
	return nil
}

func (a *Application) WaitShutdown() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	select {
	case s := <-c:
		a.logger.Info("receive shutdown signal", zap.String("signal", s.String()))
		if a.HttpServer != nil {
			if err := a.HttpServer.Stop(); err != nil {
				a.logger.Warn("failed to stop http server", zap.Error(err))
			}
		}
		if a.RpcServer != nil {
			if err := a.RpcServer.Stop(); err != nil {
				a.logger.Warn("failed to stop rpc server", zap.Error(err))
			}
		}
		os.Exit(0)
	}
}
