// Code generated by Wire. DO NOT EDIT.

//go:generate wire
//+build !wireinject

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

// Injectors from wire.go:

func BootstrapApp() (*bootstrap.Application, error) {
	config, err := vipercfg.New()
	if err != nil {
		return nil, err
	}
	options, err := zaplog.NewOptions(config)
	if err != nil {
		return nil, err
	}
	logger, err := zaplog.New(options)
	if err != nil {
		return nil, err
	}
	ginhttpOptions, err := ginhttp.NewOptions(config)
	if err != nil {
		return nil, err
	}
	ginModuleConfigurer := _wireGinModuleConfigurerValue
	engine, err := ginhttp.NewRouter(ginhttpOptions, logger, ginModuleConfigurer)
	if err != nil {
		return nil, err
	}
	server, err := ginhttp.NewServer(ginhttpOptions, logger, engine)
	if err != nil {
		return nil, err
	}
	microrpcOptions, err := microrpc.NewOptions(config, logger)
	if err != nil {
		return nil, err
	}
	microModuleConfigurer := _wireMicroModuleConfigurerValue
	service, err := microrpc.NewService(microrpcOptions, logger, microModuleConfigurer)
	if err != nil {
		return nil, err
	}
	microrpcServer, err := microrpc.NewServer(microrpcOptions, logger, service, microModuleConfigurer)
	if err != nil {
		return nil, err
	}
	application, err := bootstrap.RunApp(config, logger, server, microrpcServer)
	if err != nil {
		return nil, err
	}
	return application, nil
}

var (
	_wireGinModuleConfigurerValue   = ginhttp.GinModuleConfigurer{}
	_wireMicroModuleConfigurerValue = microrpc.MicroModuleConfigurer{}
)

// wire.go:

var providerSet = wire.NewSet(vipercfg.ProviderSet, zaplog.ProviderSet, gormdb.ProviderSet, microrpc.ProviderSet, ginhttp.ProviderSet, bootstrap.RunApp)
