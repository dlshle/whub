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
	"wsdk/relay_server/module_base"
)

type IMiddlewareManagerModule interface {
	RegisterMiddleware(middleware middleware.IServerMiddleware) error
	RunMiddlewares(conn connection.IConnection, request service.IServiceRequest) service.IServiceRequest
	Clear()
}

type MiddlewareManagerModule struct {
	*module_base.ModuleBase
	middlewares *data_structures.Tree
	logger      *logger.SimpleLogger
}

func NewMiddlewareManagerModule() IMiddlewareManagerModule {
	return &MiddlewareManagerModule{
		middlewares: data_structures.NewRedBlackTree(),
		logger:      context.Ctx.Logger().WithPrefix("[MiddlewareManagerModule]"),
	}
}

func (m *MiddlewareManagerModule) Init() error {
	m.ModuleBase = module_base.NewModuleBase("MiddlewareManager", func() error {
		var holder IMiddlewareManagerModule
		return container.Container.RemoveByType(holder)
	})
	m.middlewares = data_structures.NewRedBlackTree()
	m.logger = m.Logger()
	return container.Container.Singleton(func() IMiddlewareManagerModule {
		return m
	})
}

func (m *MiddlewareManagerModule) RunMiddlewares(conn connection.IConnection, request service.IServiceRequest) service.IServiceRequest {
	m.middlewares.ForEach(func(md interface{}) bool {
		request = md.(middleware.IServerMiddleware).Run(conn, request)
		if request.Status() > service.ServiceRequestStatusProcessing {
			return false
		}
		return true
	})
	return request
}

func (m *MiddlewareManagerModule) Clear() {
	m.middlewares.Clear()
}

func (m *MiddlewareManagerModule) RegisterMiddleware(middleware middleware.IServerMiddleware) (err error) {
	if err = middleware.Init(); err != nil {
		m.logger.Println(fmt.Sprintf("middleware %s init failed", middleware.Id()))
		return err
	}
	m.middlewares.PutKeyAsValue(middleware)
	m.logger.Println(fmt.Sprintf("middleware %s init success", middleware.Id()))
	return nil
}

func RegisterMiddleware(middleware middleware.IServerMiddleware) {
	container.Container.Call(func(manager IMiddlewareManagerModule) {
		manager.RegisterMiddleware(middleware)
	})
}

func Load() error {
	return container.Container.Singleton(func() IMiddlewareManagerModule {
		return NewMiddlewareManagerModule()
	})
}
