package middleware_manager

import (
	"fmt"
	"wsdk/common/data_structures"
	"wsdk/common/logger"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/service"
	"wsdk/relay_server/container"
	"wsdk/relay_server/context"
	"wsdk/relay_server/middleware"
)

type IMiddlewareManager interface {
	RegisterMiddleware(middleware middleware.IServerMiddleware) error
	RunMiddlewares(conn connection.IConnection, request service.IServiceRequest) service.IServiceRequest
	Clear()
}

type MiddlewareManager struct {
	middlewares *data_structures.Tree
	logger      *logger.SimpleLogger
}

func NewMiddlewareManager() IMiddlewareManager {
	return &MiddlewareManager{
		middlewares: data_structures.NewRedBlackTree(),
		logger:      context.Ctx.Logger().WithPrefix("[MiddlewareManager]"),
	}
}

func (m *MiddlewareManager) RunMiddlewares(conn connection.IConnection, request service.IServiceRequest) service.IServiceRequest {
	m.middlewares.ForEach(func(md interface{}) {
		request = md.(middleware.IServerMiddleware).Run(conn, request)
	})
	return request
}

func (m *MiddlewareManager) Clear() {
	m.middlewares.Clear()
}

func (m *MiddlewareManager) RegisterMiddleware(middleware middleware.IServerMiddleware) (err error) {
	if err = middleware.Init(); err != nil {
		m.logger.Println(fmt.Sprintf("middleware %s init failed", middleware.Id()))
		return err
	}
	m.middlewares.PutKeyAsValue(middleware)
	m.logger.Println(fmt.Sprintf("middleware %s init success", middleware.Id()))
	return nil
}

func init() {
	container.Container.Singleton(func() IMiddlewareManager {
		return NewMiddlewareManager()
	})
}
