package relay_client

import "wsdk/relay_common/service"

type ClientServiceManager struct {
	m *service.BaseServiceManager
}

type IClientServiceManager interface {
	HasService(id string) bool
	GetService(id string) IClientService
	RegisterService(string, IClientService) error
	UnregisterService(string) error
	UnregisterAllServices() error

	MatchServiceByUri(uri string) IClientService
	SupportsUri(uri string) bool
}

func (m *ClientServiceManager) HasService(id string) bool {
	return m.m.HasService(id)
}

func (m *ClientServiceManager) GetService(id string) IClientService {
	return m.m.GetService(id).(IClientService)
}

func (m *ClientServiceManager) RegisterService(id string, service IClientService) error {
	return m.m.RegisterService(id, service)
}

func (m *ClientServiceManager) UnregisterService(id string) error {
	return m.m.UnregisterService(id)
}

func (m *ClientServiceManager) UnregisterAllServices() error {
	return m.m.UnregisterAllServices()
}

func (m *ClientServiceManager) MatchServiceByUri(uri string) IClientService {
	return m.m.MatchServiceByUri(uri).(IClientService)
}

func (m *ClientServiceManager) SupportsUri(uri string) bool {
	return m.m.SupportsUri(uri)
}
