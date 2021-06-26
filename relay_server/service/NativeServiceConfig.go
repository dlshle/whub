package service

import (
	"wsdk/relay_common/service"
	"wsdk/relay_server/context"
)

type NativeServiceConfig struct {
	ctx         *context.Context
	id          string
	description string
	serviceType int
	accessType  int
	exeType     int
	handles     map[string]service.RequestHandler
}

func CreateNativeService(config NativeServiceConfig) INativeService {
	service := NewNativeService(
		config.ctx,
		config.id,
		config.description,
		config.serviceType,
		config.accessType,
		config.exeType)
	for k, v := range config.handles {
		service.RegisterRoute(k, v)
	}
	return service
}
