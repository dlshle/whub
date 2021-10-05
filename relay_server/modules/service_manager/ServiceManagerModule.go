package service_manager

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"wsdk/common/logger"
	"wsdk/common/uri_trie"
	"wsdk/common/utils"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/service"
	"wsdk/relay_server/container"
	"wsdk/relay_server/context"
	server_errors "wsdk/relay_server/errors"
	"wsdk/relay_server/events"
	"wsdk/relay_server/module_base"
	server_service "wsdk/relay_server/service_base"
)

const ServiceManagerId = "ServiceManagerModule"

type ServiceManagerModule struct {
	*module_base.ModuleBase
	// need to use full uris here!
	trieTree   *uri_trie.TrieTree
	serviceMap map[string]server_service.IService
	lock       *sync.RWMutex
	logger     *logger.SimpleLogger
}

type IServiceManagerModule interface {
	HasService(id string) bool
	GetService(id string) server_service.IService
	GetServicesByClientId(clientId string) []server_service.IService
	RegisterService(string, server_service.IService) error
	UnregisterService(string) error

	UnregisterAllServices() error
	UnregisterAllServicesFromClientId(string) error
	WithServicesFromClientId(clientId string, cb func([]server_service.IService)) error

	FindServiceByUri(uri string) server_service.IService
	MatchServiceByUri(uri string) *uri_trie.MatchContext
	SupportsUri(uri string) bool

	DescribeAllRelayServices() []service.ServiceDescriptor
	DescribeAllServices() []service.ServiceDescriptor
	UpdateService(descriptor service.ServiceDescriptor) error
}

func NewServiceManagerModule() IServiceManagerModule {
	manager := &ServiceManagerModule{
		trieTree:   uri_trie.NewTrieTree(),
		serviceMap: make(map[string]server_service.IService),
		lock:       new(sync.RWMutex),
		logger:     context.Ctx.Logger().WithPrefix("[ServiceManagerModule]"),
	}
	manager.initNotifications()
	return manager
}

func (m *ServiceManagerModule) Init() error {
	m.ModuleBase = module_base.NewModuleBase("ServiceManager", func() error {
		var holder IServiceManagerModule
		m.disposeNotifications()
		return container.Container.RemoveByType(holder)
	})
	m.trieTree = uri_trie.NewTrieTree()
	m.serviceMap = make(map[string]server_service.IService)
	m.lock = new(sync.RWMutex)
	m.logger = m.Logger()
	return container.Container.Singleton(func() IServiceManagerModule {
		return m
	})
}

func (m *ServiceManagerModule) handleServerClosed(msg messages.IMessage) {
	m.UnregisterAllServices()
}

func (m *ServiceManagerModule) initNotifications() {
	events.OnEvent(events.EventServerClosed, m.handleServerClosed)
}

func (m *ServiceManagerModule) disposeNotifications() {
	events.OffEvent(events.EventServerClosed, m.handleServerClosed)
}

func (m *ServiceManagerModule) withWrite(cb func()) {
	m.lock.Lock()
	defer m.lock.Unlock()
	cb()
}

func (s *ServiceManagerModule) UnregisterAllServices() error {
	s.logger.Println("unregister all services")
	errMsgBuilder := strings.Builder{}
	s.withWrite(func() {
		for _, service := range s.serviceMap {
			errMsgBuilder.WriteString(service.Stop().Error() + "\n")
			delete(s.serviceMap, service.Id())
		}
	})
	return errors.New(errMsgBuilder.String())
}

func (s *ServiceManagerModule) GetService(id string) server_service.IService {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.serviceMap[id]
}

func (s *ServiceManagerModule) HasService(id string) bool {
	return s.GetService(id) != nil
}

func (s *ServiceManagerModule) WithServicesFromClientId(clientId string, cb func([]server_service.IService)) error {
	services := s.GetServicesByClientId(clientId)
	cb(services)
	return nil
}

func (s *ServiceManagerModule) UnregisterAllServicesFromClientId(clientId string) error {
	s.logger.Println("unregister all services from client ", clientId)
	return s.WithServicesFromClientId(clientId, func(services []server_service.IService) {
		for i, _ := range services {
			if services[i] != nil {
				s.UnregisterService(services[i].Id())
			}
		}
	})
}

// nil -> no such client_manager, [] -> no service
func (s *ServiceManagerModule) GetServicesByClientId(id string) []server_service.IService {
	return s.getServicesByClientId(id)
}

func (s *ServiceManagerModule) getServicesByClientId(id string) []server_service.IService {
	s.lock.RLock()
	defer s.lock.RUnlock()
	var services []server_service.IService
	for _, v := range s.serviceMap {
		if v.ProviderInfo().Id == id {
			services = append(services, v)
		}
	}
	return services
}

func (s *ServiceManagerModule) serviceCountByClientId(id string) int {
	return len(s.GetServicesByClientId(id))
}

func (s *ServiceManagerModule) RegisterService(clientId string, service server_service.IService) error {
	return s.registerService(clientId, service)
}

func (s *ServiceManagerModule) registerService(clientId string, svc server_service.IService) (err error) {
	defer s.logger.Printf("register service %s from %s result: %s", svc.Id(), clientId, utils.ConditionalPick(err != nil, err, "success"))
	if clientId != context.Ctx.Server().Id() && s.serviceCountByClientId(clientId) >= server_service.MaxServicePerClient {
		err = server_errors.NewClientExceededMaxServiceCountError(clientId, server_service.MaxServicePerClient)
		return
	}
	s.withWrite(func() {
		s.serviceMap[svc.Id()] = svc
		addedPathSet := make(map[string]bool)
		for _, uri := range svc.FullServiceUris() {
			// service manager only keeps track of path, not methods, so we only add new paths
			if addedPathSet[uri] {
				continue
			}
			err = s.trieTree.Add(uri, svc, true)
			if err != nil {
				s.logger.Printf("register service(%s) route %s failed due to %s", svc.Id(), uri, err.Error())
			} else {
				addedPathSet[uri] = true
			}
		}
		/* No need to do this as when service is unregistered, all full uris will be removed
		svc.OnStopped(func(tService service.IBaseService) {
			for _, uri := range tService.FullServiceUris() {
				s.trieTree.Remove(uri)
			}
		})
		*/
	})
	err = nil
	return
}

func (s *ServiceManagerModule) UnregisterService(serviceId string) error {
	return s.unregisterService(serviceId)
}

func (s *ServiceManagerModule) unregisterService(serviceId string) error {
	svc := s.GetService(serviceId)
	if svc == nil {
		err := server_errors.NewNoSuchServiceError(serviceId)
		s.logger.Println("unregister service ", serviceId, " failed due to ", err.Error())
		return err
	}
	s.withWrite(func() {
		uris := svc.FullServiceUris()
		if s.serviceMap[serviceId] != nil {
			s.serviceMap[serviceId].Stop()
		}
		delete(s.serviceMap, serviceId)
		for _, uri := range uris {
			s.removeUriRoute(uri)
		}
	})
	s.logger.Println("unregister service ", serviceId, " succeeded")
	return nil
}

func (s *ServiceManagerModule) FindServiceByUri(uri string) server_service.IService {
	s.lock.RLock()
	defer s.lock.RUnlock()
	matchContext, err := s.trieTree.Match(uri)
	if matchContext == nil || err != nil {
		return nil
	}
	return matchContext.Value.(server_service.IService)
}

func (s *ServiceManagerModule) MatchServiceByUri(uri string) *uri_trie.MatchContext {
	s.lock.RLock()
	defer s.lock.RUnlock()
	matchContext, err := s.trieTree.Match(uri)
	if matchContext == nil || err != nil {
		return nil
	}
	return matchContext
}

func (s *ServiceManagerModule) SupportsUri(uri string) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.trieTree.SupportsUri(uri)
}

func (s *ServiceManagerModule) DescribeAllServices() []service.ServiceDescriptor {
	s.lock.RLock()
	defer s.lock.RUnlock()
	var res []service.ServiceDescriptor
	for _, v := range s.serviceMap {
		v.ServiceType()
		res = append(res, v.Describe())
	}
	return res
}

func (s *ServiceManagerModule) UpdateService(descriptor service.ServiceDescriptor) error {
	tService := s.GetService(descriptor.Id)
	if tService == nil {
		return errors.New(fmt.Sprintf("tService %s can not be found", descriptor.Id))
	}
	return utils.ProcessWithErrors(func() error {
		return tService.(*server_service.RelayService).Update(descriptor)
	}, func() error {
		// TODO how to update uris here???
		fullUris := tService.FullServiceUris()
		currentUriSet := make(map[string]bool)
		for _, uri := range fullUris {
			currentUriSet[uri] = true
		}
		for _, shortUri := range descriptor.ServiceUris {
			fullUri := fmt.Sprintf("%s/%s%s", service.ServicePrefix, descriptor.Id, shortUri)
			if !currentUriSet[fullUri] {
				// add uri_trie
				err := s.addUriRoute(tService, fullUri)
				// TODO should do atomic transaction here? revert previous steps?
				if err != nil {
					return err
				}
			} else {
				delete(currentUriSet, fullUri)
			}
		}
		for uri := range currentUriSet {
			s.removeUriRoute(uri)
		}
		return nil
	})
}

func (s *ServiceManagerModule) DescribeAllRelayServices() []service.ServiceDescriptor {
	descriptors := s.DescribeAllServices()
	// var relayDescriptors []service.ServiceDescriptor
	relayDescriptors := []service.ServiceDescriptor{}
	for _, d := range descriptors {
		if d.ServiceType == service.ServiceTypeRelay {
			relayDescriptors = append(relayDescriptors, d)
		}
	}
	return relayDescriptors
}

func (s *ServiceManagerModule) addUriRoute(service server_service.IService, route string) (err error) {
	s.withWrite(func() {
		err = s.trieTree.Add(route, service, true)
	})
	return err
}

func (s *ServiceManagerModule) removeUriRoute(route string) (success bool) {
	success = s.trieTree.Remove(route)
	return success
}

func Load() error {
	return container.Container.Singleton(func() IServiceManagerModule {
		return NewServiceManagerModule()
	})
}
