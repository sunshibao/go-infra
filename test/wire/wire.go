// +build wireinject

package main

import (
	"github.com/liuliliujian/go-infra-com/bootstrap"
	"github.com/liuliliujian/go-infra-com/config/vipercfg"
	"github.com/liuliliujian/go-infra-com/database/gormdb"
	"github.com/liuliliujian/go-infra-com/log/zaplog"
	"github.com/liuliliujian/go-infra-com/transport/http/ginhttp"
	"github.com/liuliliujian/go-infra-com/transport/rpc/microrpc"
	"github.com/google/wire"
)

var providerSet = wire.NewSet(
	vipercfg.ProviderSet,
	zaplog.ProviderSet,
	gormdb.ProviderSet,
	microrpc.ProviderSet,
	ginhttp.ProviderSet,
	bootstrap.RunApp,
	//bootstrap.RunHttpApp,
	//bootstrap.RunRpcApp,
)

func BootstrapApp() (*bootstrap.Application, error) {
	panic(wire.Build(providerSet))
}
