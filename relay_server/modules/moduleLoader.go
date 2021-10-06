package modules

import (
	"wsdk/relay_server/middleware"
	"wsdk/relay_server/module_base"
	"wsdk/relay_server/modules/auth"
	"wsdk/relay_server/modules/blocklist"
	"wsdk/relay_server/modules/client_manager"
	"wsdk/relay_server/modules/connection_manager"
	"wsdk/relay_server/modules/metering"
	"wsdk/relay_server/modules/middleware_manager"
	"wsdk/relay_server/modules/service_manager"
	"wsdk/relay_server/modules/status"
	"wsdk/relay_server/modules/throttle"
)

var middlewares []middleware.IServerMiddleware
var moduleInstances []module_base.IModule

func initModuleInstances() {
	moduleInstances = []module_base.IModule{
		new(middleware_manager.MiddlewareManagerModule),
		new(client_manager.ClientManagerModule),
		new(connection_manager.ConnectionManagerModule),
		new(auth.AuthModule),
		new(metering.MeteringModule),
		new(service_manager.ServiceManagerModule),
		new(status.ServerStatusModule),
		new(throttle.RequestThrottleModule),
		new(blocklist.BlockListModule),
	}
}

func initMiddlewares() {
	middlewares = []middleware.IServerMiddleware{
		new(connection_manager.ConnectionMiddleware),
		new(auth.AuthMiddleware),
		new(throttle.RequestAddressThrottleMiddleware),
		new(blocklist.BlockListMiddleware),
	}
}

func init() {
	initModuleInstances()
	initMiddlewares()
}

func loadCoreModules() error {
	return module_base.Manager.RegisterModules(moduleInstances)
}

func registerCoreMiddlewares() {
	middlewareManager := module_base.Manager.GetModule(middleware_manager.ID).(middleware_manager.IMiddlewareManagerModule)
	for _, m := range middlewares {
		middlewareManager.RegisterMiddleware(m)
	}
}

func InitCoreComponents() error {
	err := loadCoreModules()
	if err != nil {
		return err
	}
	registerCoreMiddlewares()
	return nil
}
