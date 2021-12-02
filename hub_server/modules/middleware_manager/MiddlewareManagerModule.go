package middleware_manager

import (
	"fmt"
	"whub/common/data_structures"
	"whub/common/logger"
	"whub/hub_common/connection"
	"whub/hub_common/service"
	"whub/hub_server/middleware"
	"whub/hub_server/module_base"
)

const ID = "MiddlewareManager"

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

func (m *MiddlewareManagerModule) Init() error {
	m.ModuleBase = module_base.NewModuleBase(ID, nil)
	m.middlewares = data_structures.NewRedBlackTree()
	m.logger = m.Logger()
	return nil
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

func RegisterMiddleware(middleware middleware.IServerMiddleware) error {
	 manager := module_base.Manager.GetModule(ID)
	 if manager == nil {
	 	return fmt.Errorf("unable to get middleware manager")
	 }
	 return manager.(IMiddlewareManagerModule).RegisterMiddleware(middleware)
}