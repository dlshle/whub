package relay_server

import (
	"errors"
	"github.com/dlshle/gommon/timed"
	"strings"
	"sync"
	"time"
	"wsdk/relay_common"
	"wsdk/relay_common/service"
)

type ServiceManager struct {
	serviceMap map[string]IServerService
	serviceExpirePeriod time.Duration
	scheduleJobPool *timed.JobPool
	lock *sync.RWMutex
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
}

func NewServiceManager(ctx *relay_common.WRContext, serviceExpirePeriod time.Duration) IServiceManager {
	return &ServiceManager{
		serviceMap: make(map[string]IServerService),
		serviceExpirePeriod: serviceExpirePeriod,
		scheduleJobPool: ctx.TimedJobPool(),
		lock: new(sync.RWMutex),
	}
}

func (m *ServiceManager) withWrite(cb func()) {
	m.lock.Lock()
	defer m.lock.RUnlock()
	cb()
}

func (s *ServiceManager) cancelTimedJob(jobId int64) bool {
	return s.scheduleJobPool.CancelJob(jobId)
}

func (s *ServiceManager) scheduleTimeoutJob(job func()) int64 {
	return s.scheduleJobPool.ScheduleAsyncTimeoutJob(job, ServiceKillTimeout)
}

func (s *ServiceManager) getServicesByClientId(id string) []IServerService {
	services := make([]IServerService, 0, MaxServicePerClient)
	s.lock.RLock()
	defer s.lock.RUnlock()
	for _, v := range s.serviceMap {
		if v.Provider().Id() == id {
			services = append(services, v)
		}
	}
	return services
}

func (s *ServiceManager) GetService(id string) IServerService {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.serviceMap[id]
}

func (s *ServiceManager) HasService(id string) bool {
	return s.GetService(id) != nil
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
				s.unregisterService(services[i].Id())
			}
		}
	})
}

// nil -> no such client, [] -> no service
func (s *ServiceManager) GetServicesByClientId(id string) []IServerService {
	return s.getServicesByClientId(id)
}

func (s *ServiceManager) serviceCountByClientId(id string) int {
	return len(s.getServicesByClientId(id))
}

func (s *ServiceManager) RegisterService(clientId string, service IServerService) error {
	return s.registerService(clientId, service)
}

// assume server has ensured client id exists
func (s *ServiceManager) registerService(clientId string, service IServerService) error {
	if s.serviceCountByClientId(clientId) >= MaxServicePerClient {
		return NewClientExceededMaxServiceCountError(clientId)
	}
	var serviceDeadTimeoutJobId int64 = -1
	s.withWrite(func() {
		service.OnHealthCheckFails(func(service IServerService) {
			// log
			service.KillAllProcessingJobs()
			// schedule timeout job to really kill the service if it's been dead for X duration
			serviceDeadTimeoutJobId = s.scheduleTimeoutJob(func() {
				s.unregisterService(service.Id())
			})
		})
		service.OnHealthRestored(func(service IServerService) {
			// log
			if serviceDeadTimeoutJobId > -1 {
				s.cancelTimedJob(serviceDeadTimeoutJobId)
			}
		})
		s.serviceMap[service.Id()] = service
	})
	return nil
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

func (s *ServiceManager) UnregisterService(serviceId string) error {
	return s.unregisterService(serviceId)
}

func (s *ServiceManager) unregisterService(serviceId string) error {
	if !s.HasService(serviceId) {
		return NewNoSuchServiceError(serviceId)
	}
	s.withWrite(func() {
		s.serviceMap[serviceId].Stop()
		delete(s.serviceMap, serviceId)
	})
	return nil
}

func (s *ServiceManager) MatchServiceByUri(uri string) IServerService {
	if !strings.HasPrefix(uri, service.ServicePrefix) {
		return nil
	}
	s.lock.RLock()
	defer s.lock.RUnlock()
	for _, v := range s.serviceMap {
		if v.SupportsUri(uri) {
			return v
		}
	}
	return nil
}

