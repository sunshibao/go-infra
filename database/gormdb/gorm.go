package gormdb

import (
	"github.com/liuliliujian/go-infra-com/config"
	"errors"
	"github.com/google/wire"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	errs "github.com/pkg/errors"
	"github.com/wantedly/gorm-zap"
	"go.uber.org/zap"
	"strings"
	"time"
)

type Options struct {
	URL               string
	MaxConns          int
	MaxIdleConns      int
	MaxConnLifetime   time.Duration
	BlockGlobalUpdate bool
	Debug             bool
}

func NewOptions(c config.Config, logger *zap.Logger) (*Options, error) {
	o := &Options{
		MaxConns:          10,
		MaxIdleConns:      2,
		MaxConnLifetime:   time.Hour,
		BlockGlobalUpdate: true,
		Debug:             false,
	}
	if err := c.UnmarshalKey("db", o); err != nil {
		return nil, errs.WithMessage(err, "invalid db config options")
	}
	if strings.TrimSpace(o.URL) == "" {
		return nil, errors.New("database url config option must be supplied")
	}
	logger.Info("load database options success", zap.String("url", mysqlCoveredURL(o)))
	return o, nil
}

func New(o *Options, logger *zap.Logger) (*gorm.DB, error) {
	//todo 考虑如何将dialect提取成配置，以便通用，那样的话相应db的dialect需要用户自行加载，有没有办法动态加载???
	db, err := gorm.Open("mysql", o.URL)
	if err != nil {
		return nil, errs.WithMessage(err, "failed to open db connection")
	}
	if o.Debug {
		db.LogMode(true)
	}
	db.SetLogger(gormzap.New(logger))
	db.SingularTable(true)
	if o.BlockGlobalUpdate {
		db.BlockGlobalUpdate(true)
	}
	db.DB().SetMaxOpenConns(o.MaxConns)
	db.DB().SetMaxIdleConns(o.MaxIdleConns)
	db.DB().SetConnMaxLifetime(o.MaxConnLifetime)
	//todo add monitor middleware
	//notice: graceful close db pool in app's code
	return db, nil
}

func mysqlCoveredURL(o *Options) string {
	colonIdx := strings.Index(o.URL, ":")
	if colonIdx == -1 {
		return o.URL
	}
	return o.URL[:colonIdx] + ":***" + o.URL[colonIdx + 1:]
}

var ProviderSet = wire.NewSet(New, NewOptions)