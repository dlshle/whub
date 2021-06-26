package controllers

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"wsdk/common/timed"
	"wsdk/relay_common/service"
	"wsdk/relay_common/uri"
	"wsdk/relay_common/utils"
	"wsdk/relay_server/context"
	servererror "wsdk/relay_server/errors"
	service2 "wsdk/relay_server/service"
)

const ServiceManagerId = "ServiceManager"

type ServiceManager struct {
	// need to use full uris here!
	trieTree        *uri.TrieTree
	serviceMap      map[string]service2.IService
	scheduleJobPool *timed.JobPool
	lock            *sync.RWMutex
}

type IServiceManager interface {
	Id() string
	HasService(id string) bool
	GetService(id string) service2.IService
	GetServicesByClientId(clientId string) []service2.IService
	RegisterService(string, service2.IService) error
	UnregisterService(string) error

	UnregisterAllServices() error
	UnregisterAllServicesFromClientId(string) error
	WithServicesFromClientId(clientId string, cb func([]service2.IService)) error

	FindServiceByUri(uri string) service2.IService
	SupportsUri(uri string) bool

	UpdateService(descriptor service.ServiceDescriptor) error
}

func NewServiceManager(ctx *context.Context) IServiceManager {
	return &ServiceManager{
		trieTree:        uri.NewTrieTree(),
		serviceMap:      make(map[string]service2.IService),
		scheduleJobPool: ctx.TimedJobPool(),
		lock:            new(sync.RWMutex),
	}
}

func (m *ServiceManager) withWrite(cb func()) {
	m.lock.Lock()
	defer m.lock.RUnlock()
	cb()
}

func (s *ServiceManager) Id() string {
	return ServiceManagerId
}

func (s *ServiceManager) UnregisterAllServices() error {
	errMsgBuilder := strings.Builder{}
	s.withWrite(func() {
		for _, service := range s.serviceMap {
			errMsgBuilder.WriteString(service.Stop().Error() + "\n")
			delete(s.serviceMap, service.Id())
		}
	})
	return errors.New(errMsgBuilder.String())
}

func (s *ServiceManager) GetService(id string) service2.IService {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.serviceMap[id]
}

func (s *ServiceManager) HasService(id string) bool {
	return s.HasService(id)
}

func (s *ServiceManager) WithServicesFromClientId(clientId string, cb func([]service2.IService)) error {
	services := s.GetServicesByClientId(clientId)
	cb(services)
	return nil
}

func (s *ServiceManager) UnregisterAllServicesFromClientId(clientId string) error {
	return s.WithServicesFromClientId(clientId, func(services []service2.IService) {
		for i, _ := range services {
			if services[i] != nil {
				s.UnregisterService(services[i].Id())
			}
		}
	})
}

// nil -> no such client, [] -> no service
func (s *ServiceManager) GetServicesByClientId(id string) []service2.IService {
	return s.getServicesByClientId(id)
}

func (s *ServiceManager) getServicesByClientId(id string) []service2.IService {
	s.lock.RLock()
	defer s.lock.RUnlock()
	var services []service2.IService
	for _, v := range s.serviceMap {
		if v.ProviderInfo().Id == id {
			services = append(services, v)
		}
	}
	return services
}

func (s *ServiceManager) serviceCountByClientId(id string) int {
	return len(s.GetServicesByClientId(id))
}

func (s *ServiceManager) RegisterService(clientId string, service service2.IService) error {
	return s.registerService(clientId, service)
}

func (s *ServiceManager) registerService(clientId string, svc service2.IService) error {
	if s.serviceCountByClientId(clientId) >= service2.MaxServicePerClient {
		return servererror.NewClientExceededMaxServiceCountError(clientId, service2.MaxServicePerClient)
	}
	s.withWrite(func() {
		s.serviceMap[svc.Id()] = svc
		for _, uri := range svc.FullServiceUris() {
			s.trieTree.Add(uri, svc, false)
		}
		svc.OnStopped(func(tService service.IBaseService) {
			for _, uri := range tService.FullServiceUris() {
				s.trieTree.Remove(uri)
			}
		})
	})
	return nil
}

func (s *ServiceManager) UnregisterService(serviceId string) error {
	return s.unregisterService(serviceId)
}

func (s *ServiceManager) unregisterService(serviceId string) error {
	if !s.HasService(serviceId) {
		// TODO use predefined service errors
		return errors.New("no such service " + serviceId)
	}
	s.withWrite(func() {
		s.serviceMap[serviceId].Stop()
		delete(s.serviceMap, serviceId)
	})
	return nil
}

func (s *ServiceManager) FindServiceByUri(uri string) service2.IService {
	s.lock.RLock()
	defer s.lock.RUnlock()
	matchContext, err := s.trieTree.Match(uri)
	if matchContext == nil || err != nil {
		return nil
	}
	return matchContext.Value.(service2.IService)
}

func (s *ServiceManager) SupportsUri(uri string) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.trieTree.SupportsUri(uri)
}

func (s *ServiceManager) UpdateService(descriptor service.ServiceDescriptor) error {
	tService := s.GetService(descriptor.Id)
	if tService == nil {
		return errors.New(fmt.Sprintf("tService %s can not be found", descriptor.Id))
	}
	return utils.ProcessWithErrors(func() error {
		return tService.(service2.RelayService).Update(descriptor)
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
				err := s.addUriRoute(tService, fullUri)
				// TODO should do atomic transaction here? revert previous steps?
				if err != nil {
					return err
				}
				delete(currentUriSet, fullUri)
			}
		}
		for uri := range currentUriSet {
			s.removeUriRoute(uri)
		}
		return nil
	})
}

func (s *ServiceManager) cancelTimedJob(jobId int64) bool {
	return s.scheduleJobPool.CancelJob(jobId)
}

func (s *ServiceManager) scheduleTimeoutJob(job func()) int64 {
	return s.scheduleJobPool.ScheduleAsyncTimeoutJob(job, service2.ServiceKillTimeout)
}

func (s *ServiceManager) addUriRoute(service service2.IService, route string) (err error) {
	s.withWrite(func() {
		err = s.trieTree.Add(route, service, true)
	})
	return err
}

func (s *ServiceManager) removeUriRoute(route string) (success bool) {
	s.withWrite(func() {
		success = s.trieTree.Remove(route)
	})
	return success
}
