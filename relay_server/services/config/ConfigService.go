package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"wsdk/common/logger"
	"wsdk/relay_common/service"
	"wsdk/relay_server/container"
	"wsdk/relay_server/core/config"
	"wsdk/relay_server/service_base"
)

const (
	ID          = "config"
	RouteGet    = "/get/:key"
	RouteGetAll = "/get"
	RouteSet    = "/set"
)

type ConfigService struct {
	*service_base.NativeService
	manager config.IServerConfigManager `$inject:""`
	logger  *logger.SimpleLogger
}

func (s *ConfigService) Init() (err error) {
	s.NativeService = service_base.NewNativeService(ID, "basic messaging service", service.ServiceTypeInternal, service.ServiceAccessTypeBoth, service.ServiceExecutionSync)
	err = container.Container.Fill(s)
	if err != nil {
		return err
	}
	routeMap := make(map[string]service.RequestHandler)
	routeMap[RouteGetAll] = s.GetAll
	routeMap[RouteGet] = s.Get
	routeMap[RouteSet] = s.Set
	return s.InitRoutes(routeMap)
}

func (s *ConfigService) GetAll(request service.IServiceRequest, pathParams map[string]string, queryParams map[string]string) (err error) {
	configs := s.manager.GetConfigs()
	marshalled, err := json.Marshal(configs)
	if err != nil {
		return
	}
	s.ResolveByResponse(request, marshalled)
	return
}

func (s *ConfigService) Get(request service.IServiceRequest, pathParams map[string]string, queryParams map[string]string) (err error) {
	configKey := pathParams["key"]
	config := s.manager.GetConfig(configKey)
	if config == nil {
		return NewInvalidConfigKeyError(configKey)
	}
	marshalled, err := json.Marshal(config)
	if err != nil {
		return
	}
	s.ResolveByResponse(request, marshalled)
	return
}

func (s *ConfigService) Set(request service.IServiceRequest, pathParams map[string]string, queryParams map[string]string) (err error) {
	return errors.New("unsupported")
}

func NewInvalidConfigKeyError(key string) error {
	return errors.New(fmt.Sprintf("invalid config key %s", key))
}
