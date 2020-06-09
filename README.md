Infra common
-------
技术组件模块化，开箱即用，基于配置(Wire)自动装配，提供自定义扩展入口，消除服务化项目中的代码冗余，并在通用组件中织入治理与监控等，帮助研发关注于业务.

gateway ---> api ---> service

## Features

* Gin Http Server

* Micro Rpc Client & Server

* DB Gorm

* Viper Config

* Zap Log

* Stack Dumper

* Todo List

  * base repository
  
  * api result protocol & general dev flow

  * api swagger
  
  * oauth2.0 token auth
  
  * service chain monitor
  
  * config center
  
  * service hystrix
  
  * service layer tx
  
  * job scheduler
  
  * table shard & aggregate
  
  * redis client
  
  * mq client
  
  * devops(docker & k8s)
  
  * ...

## Quick start
```go
//wire.go
var providerSet = wire.NewSet(
	vipercfg.ProviderSet,
	zaplog.ProviderSet,
	gormdb.ProviderSet,
	microrpc.ProviderSet,
	ginhttp.ProviderSet,
	bootstrap.RunApp,       //run http & rpc server, choose one
	bootstrap.RunHttpApp,   //run http server, choose one
	bootstrap.RunRpcApp,    //run rpc server, choose one
)

func BootstrapApp() (*bootstrap.Application, error) {
	panic(wire.Build(providerSet))
}

//main.go
func main() {
	app, err := BootstrapApp()
	if err != nil {
		panic(err)
	}
	app.WaitShutdown()
}

//execute wire to generate wire_gen.go, then run main.go
```
