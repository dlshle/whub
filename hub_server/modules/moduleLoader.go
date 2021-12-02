package modules

import (
	"whub/hub_server/middleware"
	"whub/hub_server/module_base"
	"whub/hub_server/modules/auth"
	"whub/hub_server/modules/blocklist"
	"whub/hub_server/modules/client_manager"
	"whub/hub_server/modules/connection_manager"
	"whub/hub_server/modules/metering"
	"whub/hub_server/modules/middleware_manager"
	"whub/hub_server/modules/service_manager"
	"whub/hub_server/modules/status"
	"whub/hub_server/modules/throttle"
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
		new(blocklist.BlockListModule),
		new(throttle.RequestThrottleModule),
	}
}

// middleware registration was moved to the Init of each Module
/*
func initMiddlewares() {
	middlewares = []middleware.IServerMiddleware{
		new(connection_manager.ConnectionMiddleware),
		new(auth.AuthMiddleware),
		new(throttle.RequestAddressThrottleMiddleware),
		new(blocklist.BlockListMiddleware),
	}
}
 */

func init() {
	initModuleInstances()
	// initMiddlewares()
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
