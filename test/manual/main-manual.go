package main

import (
	config2 "github.com/liuliliujian/go-infra-com/config"
	"github.com/liuliliujian/go-infra-com/config/vipercfg"
	"github.com/liuliliujian/go-infra-com/database/gormdb"
	"github.com/liuliliujian/go-infra-com/log/zaplog"
	"github.com/liuliliujian/go-infra-com/transport/rpc/microrpc"
	"errors"
	"fmt"
	"github.com/asaskevich/govalidator"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

func main() {
	var err error
	fmt.Println(err == nil)
	pflag.Set("config", "test/conf.yml")
	config, err := vipercfg.New()
	if err != nil {
		panic(err)
	}
	fmt.Println(config2.GetApplicationName(config), config.GetString("ccc") == "")

	fmt.Println(config.Get("zap-logs"))
	o := zaplog.Options {Options: []zaplog.Option {}}
	fmt.Println("o:", o.Options == nil, len(o.Options), config.IsSet("zap-logs"))
	err = config.UnmarshalKey("zap-logs", &(o.Options))
	if err != nil {
		panic(err)
	}
	fmt.Println(o)

	options := make([]zaplog.Option, 0, 10)
	err = config.UnmarshalKey("zap-logs", &options)
	if err != nil {
		panic(err)
	}
	fmt.Println(options)

	isPath, i := govalidator.IsFilePath("./a.log"[1:])
	fmt.Println(isPath, i)

	zapOptions, err := zaplog.NewOptions(config)
	if err != nil {
		panic(err)
	}
	fmt.Println(*zapOptions)

	logger, err := zaplog.New(zapOptions)
	checkError(err)
	logger.Error("error some message", zap.String("ke", "ve"), zap.Error(errors.New("some biz error")))
	logger.Debug("debug some message")
	logger.Info("info some message", zap.String("ki", "vi"))
	logger.Warn("warn some message", zap.String("kw", "vw"), zap.String("ts", "vw"), zap.String("time", "vw"))
	logger.Named("test.main").Info("info named some message", zap.String("ki", "vi"))
	logger.Info("gorm", )

	gormOptions, err := gormdb.NewOptions(config, logger)
	db, err := gormdb.New(gormOptions, logger)
	checkError(err)
	db.Exec("update car set price = price + ?", 1)

	microOptions, err := microrpc.NewOptions(config, logger)
	checkError(err)
	fmt.Printf("micro options:%+v, %+v, %+v\n", *microOptions, microOptions.Server, microOptions.Client)
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
