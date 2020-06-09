package config

import (
	"github.com/spf13/pflag"
	"math/rand"
	"time"
)

//项目统一使用pflag
var _ = pflag.String(CONF_FILE, "", "config file")
var _ = pflag.String(CONF_ENV, ENV_DEV, "run environment")

func init() {
	//先放在这里，config最先被加载
	rand.Seed(time.Now().UnixNano())
}

type Config interface {
	Get(key string) interface{}
	GetString(key string) string
	GetBool(key string) bool
	GetInt(key string) int
	GetInt32(key string) int32
	GetInt64(key string) int64
	GetFloat64(key string) float64
	GetTime(key string) time.Time
	GetDuration(key string) time.Duration
	GetStringSlice(key string) []string
	GetStringMap(key string) map[string]interface{}
	GetStringMapString(key string) map[string]string
	GetStringMapStringSlice(key string) map[string][]string
	GetSizeInBytes(key string) uint
	IsSet(key string) bool
	Sub(key string) Config
	Unmarshal(obj interface{}) error
	UnmarshalKey(key string, obj interface{}) error
}

//type ConfigPath string
//
//type Environment string

const (
	CONF_FILE      = "config"
	CONF_ENV       = "env"
	CONF_APP_PREF  = "application"
	CONF_APPNAME   = CONF_APP_PREF + ".name"
	CONF_STACKDUMP = CONF_APP_PREF + ".stackDump"

	ENV_DEV   = "dev"
	ENV_DAILY = "daily"
	ENV_TEST  = "test"
	ENV_PRE   = "pre"
	ENV_PROD  = "prod"
)

func GetApplicationName(conf Config) string {
	return conf.GetString(CONF_APPNAME)
}

func GetEnvironment(conf Config) string {
	return conf.GetString(CONF_ENV)
}
