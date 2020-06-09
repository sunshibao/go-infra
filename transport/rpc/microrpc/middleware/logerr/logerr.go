package logerr

import (
	"context"
	"fmt"
	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/server"
	"go.uber.org/zap"
	"strings"
)

/*
	micro service call调用出错打印日志，避免每次需要开发人员代码额外打印
*/
type clientWrapper struct {
	client.Client
	logger *zap.Logger
}

func (c *clientWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	err := c.Client.Call(ctx, req, rsp, opts...)
	if err != nil {
		service := req.Service()
		handler := "unknown"	//保险一点避免出错
		method := "unkown"
		fragments := strings.Split(req.Method(), ".")
		if len(fragments) > 0 {
			handler = fragments[0] + "Service"
		}
		if len(fragments) > 1 {
			method = fragments[1]
		}
		c.logger.Error(fmt.Sprintf("call micro service[%s:%s.%s] failed", service, handler, method), zap.Error(err))
	}
	return err
}

func NewClientWrapper(logger *zap.Logger) client.Wrapper {
	return func(c client.Client) client.Client {
		return &clientWrapper{
			Client: c,
			logger: logger,
		}
	}
}

func NewHandlerWrapper(logger *zap.Logger) server.HandlerWrapper {
	return func(handlerFunc server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			err := handlerFunc(ctx, req, rsp)
			if err != nil {
				service := req.Service()
				handler := "unknown"
				method := "unkown"
				fragments := strings.Split(req.Method(), ".")
				if len(fragments) > 0 {
					handler = fragments[0] + "Service"
				}
				if len(fragments) > 1 {
					method = fragments[1]
				}
				logger.Error(fmt.Sprintf("handle micro request[%s:%s.%s] failed", service, handler, method), zap.Error(err))
			}
			return err
		}
	}
}
