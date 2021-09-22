package core

import (
	"wsdk/common/utils"
	"wsdk/relay_server/container"
	"wsdk/relay_server/core/auth"
	"wsdk/relay_server/core/client_manager"
	"wsdk/relay_server/core/connection_manager"
	"wsdk/relay_server/core/metering"
	"wsdk/relay_server/core/middleware_manager"
	"wsdk/relay_server/core/service_manager"
	"wsdk/relay_server/core/status"
	"wsdk/relay_server/core/throttle"
	"wsdk/relay_server/middleware"
)

var moduleLoaders []func() error
var middlewares []middleware.IServerMiddleware

func initModuleLoaders() {
	moduleLoaders = []func() error{
		middleware_manager.Load,
		client_manager.Load,
		connection_manager.Load,
		auth.Load,
		metering.Load,
		service_manager.Load,
		status.Load,
		throttle.Load,
	}
}

func initMiddlewares() {
	middlewares = []middleware.IServerMiddleware{
		new(connection_manager.ConnectionMiddleware),
		new(auth.AuthMiddleware),
		new(throttle.RequestAddressThrottleMiddleware),
	}
}

func init() {
	initModuleLoaders()
	initMiddlewares()
}

func LoadCoreModules() error {
	err := utils.ProcessWithError(moduleLoaders)
	if err != nil {
		container.Container.Reset()
		return err
	}
	return nil
}

func RegisterCoreMiddlewares() {
	for _, m := range middlewares {
		container.Container.Call(func(manager middleware_manager.IMiddlewareManager) {
			manager.RegisterMiddleware(m)
		})
	}
}

func InitCoreComponents() error {
	err := LoadCoreModules()
	if err != nil {
		return err
	}
	RegisterCoreMiddlewares()
	return nil
}
