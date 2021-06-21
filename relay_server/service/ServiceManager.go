package service

import (
	"errors"
	"fmt"
	"wsdk/relay_common"
	"wsdk/relay_common/service"
	"wsdk/relay_common/utils"
	"wsdk/relay_server"
)

type ServiceManager struct {
	// need to use full uris here!
	m *service.BaseServiceManager
}

type IServiceManager interface {
	HasService(id string) bool
	GetService(id string) IServerService
	GetServicesByClientId(clientId string) []IServerService
	RegisterService(string, IServerService) error
	UnregisterService(string) error

	UnregisterAllServices() error
	UnregisterAllServicesFromClientId(string) error
	WithServicesFromClientId(clientId string, cb func([]IServerService)) error

	MatchServiceByUri(uri string) IServerService
	SupportsUri(uri string) bool

	UpdateService(descriptor service.ServiceDescriptor) error
}

func NewServiceManager(ctx *relay_common.WRContext) IServiceManager {
	return &ServiceManager{service.NewBaseServiceManager(ctx)}
}

func (s *ServiceManager) UnregisterAllServices() error {
	return s.m.UnregisterAllServices()
}

func (s *ServiceManager) getServicesByClientId(id string) []IServerService {
	bServices := s.m.MatchServicesBy(func(service service.IBaseService) bool {
		return service.ProviderInfo().Id == id
	})
	services := make([]IServerService, len(bServices))
	for i := range bServices {
		services[i] = bServices[i].(IServerService)
	}
	return services
}

func (s *ServiceManager) GetService(id string) IServerService {
	return s.m.GetService(id).(IServerService)
}

func (s *ServiceManager) HasService(id string) bool {
	return s.HasService(id)
}

func (s *ServiceManager) WithServicesFromClientId(clientId string, cb func([]IServerService)) error {
	services := s.GetServicesByClientId(clientId)
	cb(services)
	return nil
}

func (s *ServiceManager) UnregisterAllServicesFromClientId(clientId string) error {
	return s.WithServicesFromClientId(clientId, func(services []IServerService) {
		for i, _ := range services {
			if services[i] != nil {
				s.UnregisterService(services[i].Id())
			}
		}
	})
}

// nil -> no such client, [] -> no service
func (s *ServiceManager) GetServicesByClientId(id string) []IServerService {
	return s.getServicesByClientId(id)
}

func (s *ServiceManager) serviceCountByClientId(id string) int {
	return len(s.GetServicesByClientId(id))
}

func (s *ServiceManager) RegisterService(clientId string, service IServerService) error {
	if s.serviceCountByClientId(clientId) >= relay_server.MaxServicePerClient {
		return relay_server.NewClientExceededMaxServiceCountError(clientId)
	}
	return s.m.RegisterService(clientId, service)
}

func (s *ServiceManager) UnregisterService(serviceId string) error {
	return s.m.UnregisterService(serviceId)
}

func (s *ServiceManager) MatchServiceByUri(uri string) IServerService {
	return s.m.MatchServiceByUri(uri).(IServerService)
}

func (s *ServiceManager) SupportsUri(uri string) bool {
	return s.m.SupportsUri(uri)
}

func (s *ServiceManager) UpdateService(descriptor service.ServiceDescriptor) error {
	tService := s.GetService(descriptor.Id)
	if tService == nil {
		return errors.New(fmt.Sprintf("tService %s can not be found", descriptor.Id))
	}
	return utils.ProcessWithErrors(func() error {
		return tService.Update(descriptor)
	}, func() error {
		// TODO how to update uris here???
		fullUris := tService.FullServiceUris()
		currentUriSet := make(map[string]bool)
		for _, uri := range fullUris {
			currentUriSet[uri] = true
		}
		for _, shortUri := range descriptor.ServiceUris {
			fullUri := fmt.Sprintf("%s/%s", service.ServicePrefix, shortUri)
			if !currentUriSet[fullUri] {
				// add uri
				err := s.m.AddUriRoute(tService, fullUri)
				// TODO should do atomic transaction here? revert previous steps?
				if err != nil {
					return err
				}
				delete(currentUriSet, fullUri)
			}
		}
		for uri := range currentUriSet {
			s.m.RemoveUriRoute(uri)
		}
		return nil
	})
}
