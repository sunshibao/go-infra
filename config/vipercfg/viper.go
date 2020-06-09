package vipercfg

import (
	"github.com/liuliliujian/go-infra-com/config"
	"fmt"
	"github.com/google/wire"
	errs "github.com/pkg/errors"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	_ "github.com/spf13/viper/remote"
	"strings"
	"time"
)

type viperConfig struct {
	*viper.Viper
}

func (v *viperConfig) Sub(key string) config.Config {
	return &viperConfig{Viper: v.Viper.Sub(key)}
}

func (v *viperConfig) Unmarshal(obj interface{}) error {
	return v.Viper.Unmarshal(obj)
}

func (v *viperConfig) UnmarshalKey(key string, obj interface{}) error {
	return v.Viper.UnmarshalKey(key, obj)
}

/*
todo: 先使用viper，后续考虑集成配置中心
Priority:
	explicit call to Set
	flag
	env
	config
	key/value store
	default
*/
func New() (config.Config, error) {
	var (
		v   = viper.New()
	)
	v.AutomaticEnv()
	v.BindPFlags(pflag.CommandLine)
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))

	env := v.GetString(config.CONF_ENV)
	configFile := v.GetString(config.CONF_FILE)
	if configFile != "" {
		v.SetConfigFile(configFile)
		if err := v.ReadInConfig(); err != nil {
			return nil, errs.WithMessagef(err, "failed to load config file:%s\n", configFile)
		}
	} else {
		v.AddConfigPath(".")
		v.AddConfigPath("config/")
		v.AddConfigPath("conf/")
		v.SetConfigName("conf-" + env)
		if err := v.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				v.SetConfigName("conf")
				if err := v.ReadInConfig(); err != nil {
					if _, ok := err.(viper.ConfigFileNotFoundError); ok {
						return nil, errs.New("config file not found")
					}
					return nil, errs.WithMessagef(err, "failed to load config file:%s\n", v.ConfigFileUsed())
				}
			} else {
				return nil, errs.WithMessagef(err, "failed to load config file:%s\n", v.ConfigFileUsed())
			}
		}
	}
	fmt.Printf("read config file: %s\n", v.ConfigFileUsed())

	if env == config.ENV_DEV {
		v.WatchConfig()
	}

	if v.IsSet("config.etcd.addrs") && v.IsSet("config.etcd.path") {
		etcdAddrs := v.GetStringSlice("config.etcd.addrs")
		etcdCfgPath := v.GetString("config.etcd.path")
		for _, etcdAddr := range etcdAddrs {
			v.AddRemoteProvider("etcd", etcdAddr, etcdCfgPath)
		}
		etcdCfgType := "yml"
		if v.IsSet("config.etcd.type") {
			etcdCfgType = v.GetString("config.etcd.type")
		} else if strings.Contains(etcdCfgPath, ".") {
			etcdCfgType = etcdCfgPath[strings.LastIndex(etcdCfgPath, ".")+1:]
		}
		v.GetString(strings.TrimSpace(etcdCfgType))
		err := v.ReadRemoteConfig()
		if err != nil {
			return nil, err
		}
		fmt.Printf("read remote etcd config: %s, %s, %s\n", etcdAddrs, etcdCfgPath, etcdCfgType)

		if env == config.ENV_DEV {
			for {
				time.Sleep(5 * time.Second)
				err := v.WatchRemoteConfig()
				if err != nil {
					fmt.Printf("failed to read remote config")
					continue
				}
			}
		}
	}

	return &viperConfig{Viper: v}, nil
}

var ProviderSet = wire.NewSet(New)
