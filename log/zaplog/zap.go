package zaplog

import (
	"errors"
	"fmt"
	"github.com/asaskevich/govalidator"
	"github.com/google/wire"
	"github.com/liuliliujian/go-infra-com/config"
	"github.com/natefinch/lumberjack"
	errs "github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"
)

type Options struct {
	Options []Option
	AppName string
}

type Option struct {
	FilePath   string
	MaxSize    int
	MaxBackups int
	MaxAge     int
	Level      string
	Lv         zapcore.Level
}

func NewOptions(c config.Config) (*Options, error) {
	var (
		o = Options{Options: make([]Option, 0, 3)}
	)

	if err := c.UnmarshalKey("zap-logs", &o.Options); err != nil {
		return nil, errs.WithMessage(err, "invalid zap log config options")
	}

	for idx, _ := range o.Options {
		option := &o.Options[idx]
		if option.FilePath == "" {
			return nil, errors.New("log file path cannot be empty")
		}
		if option.FilePath != "stdout" && option.FilePath != "stderr" {
			fpath := option.FilePath
			if strings.HasPrefix(fpath, ".") {
				fpath = fpath[1:]
			}
			if isFilePath, _ := govalidator.IsFilePath(fpath); !isFilePath {
				return nil, errors.New(fmt.Sprintf("log file path[%s] is invalid", option.FilePath))
			}
		}
		var lv zapcore.Level
		if err := lv.UnmarshalText([]byte(option.Level)); err != nil {
			return nil, errors.New(fmt.Sprintf("log level[%s] is invalid", option.Level))
		}
		option.Lv = lv
	}
	o.AppName = config.GetApplicationName(c)
	return &o, nil
}

func New(o *Options) (*zap.Logger, error) {
	var (
		logger *zap.Logger
	)

	var ew zapcore.WriteSyncer

	cores := make([]zapcore.Core, 0, 5)
	for _, option := range o.Options {
		developmentEncoderConfig := zap.NewDevelopmentEncoderConfig()
		//todo 临时添加，添加日志caller func，后面梳理，包括ProductionEncoder
		developmentEncoderConfig.EncodeCaller = callerEncoder
		if option.FilePath == "stdout" {
			ce := zapcore.NewConsoleEncoder(developmentEncoderConfig)
			cores = append(cores, zapcore.NewCore(ce, zapcore.Lock(os.Stdout), zap.NewAtomicLevelAt(option.Lv)))
		} else if option.FilePath == "stderr" {
			ew = zapcore.Lock(os.Stderr)
			ce := zapcore.NewConsoleEncoder(developmentEncoderConfig)
			cores = append(cores, zapcore.NewCore(ce, ew, zap.NewAtomicLevelAt(option.Lv)))
		} else {
			fw := zapcore.AddSync(&lumberjack.Logger{
				Filename:   option.FilePath,
				MaxSize:    option.MaxSize, // megabytes
				MaxBackups: option.MaxBackups,
				MaxAge:     option.MaxAge, // days
				LocalTime:  true,
			})
			productionEncoderConfig := zap.NewProductionEncoderConfig()
			//添加下划线前缀, 避免与biz fields重复
			productionEncoderConfig.TimeKey = "_time"
			productionEncoderConfig.LevelKey = "_level"
			productionEncoderConfig.NameKey = "_logger"
			productionEncoderConfig.CallerKey = "_caller"
			productionEncoderConfig.MessageKey = "_msg"
			productionEncoderConfig.StacktraceKey = "_stacktrace"
			productionEncoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
				enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
			}
			je := zapcore.NewJSONEncoder(productionEncoderConfig)
			cores = append(cores, zapcore.NewCore(je, fw, zap.NewAtomicLevelAt(option.Lv)))
		}
	}

	if ew == nil {
		ew = zapcore.Lock(os.Stderr)
	}

	core := zapcore.NewTee(cores...)

	initialFields := make(map[string]interface{})
	if o.AppName != "" {
		initialFields["app"] = o.AppName
	}

	cfg := zap.Config{
		Development:       false,
		DisableCaller:     false,
		DisableStacktrace: false,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		InitialFields: initialFields,
	}

	logger = zap.New(core, buildOptions(cfg, ew)...)
	zap.ReplaceGlobals(logger)

	return logger, nil
}

func callerEncoder(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	funcName := runtime.FuncForPC(caller.PC).Name()
	slashLastIdx := strings.LastIndex(funcName, "/")
	enc.AppendString(strings.Join([]string{caller.TrimmedPath(), funcName[slashLastIdx + 1:]}, ":"))
}

func buildOptions(cfg zap.Config, errSink zapcore.WriteSyncer) []zap.Option {
	opts := []zap.Option{zap.ErrorOutput(errSink)}

	if cfg.Development {
		opts = append(opts, zap.Development())
	}

	if !cfg.DisableCaller {
		opts = append(opts, zap.AddCaller())
	}

	stackLevel := zap.ErrorLevel
	if cfg.Development {
		stackLevel = zap.WarnLevel
	}
	if !cfg.DisableStacktrace {
		opts = append(opts, zap.AddStacktrace(stackLevel))
	}

	if cfg.Sampling != nil {
		opts = append(opts, zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return zapcore.NewSampler(core, time.Second, int(cfg.Sampling.Initial), int(cfg.Sampling.Thereafter))
		}))
	}

	if len(cfg.InitialFields) > 0 {
		fs := make([]zap.Field, 0, len(cfg.InitialFields))
		keys := make([]string, 0, len(cfg.InitialFields))
		for k := range cfg.InitialFields {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fs = append(fs, zap.Any(k, cfg.InitialFields[k]))
		}
		opts = append(opts, zap.Fields(fs...))
	}

	return opts
}

var ProviderSet = wire.NewSet(New, NewOptions)
