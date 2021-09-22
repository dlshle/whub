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
	m.middlewares.ForEach(func(md interface{}) bool {
		request = md.(middleware.IServerMiddleware).Run(conn, request)
		if request.Status() > service.ServiceRequestStatusProcessing {
			return false
		}
		return true
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

func RegisterMiddleware(middleware middleware.IServerMiddleware) {
	container.Container.Call(func(manager IMiddlewareManager) {
		manager.RegisterMiddleware(middleware)
	})
}

func Load() error {
	return container.Container.Singleton(func() IMiddlewareManager {
		return NewMiddlewareManager()
	})
}
