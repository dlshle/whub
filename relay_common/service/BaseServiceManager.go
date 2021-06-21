package service

import (
	"errors"
	"strings"
	"sync"
	"wsdk/common/timed"
	"wsdk/relay_common"
	"wsdk/relay_common/uri"
	"wsdk/relay_server"
)

type BaseServiceManager struct {
	trieTree        *uri.TrieTree
	serviceMap      map[string]IBaseService
	scheduleJobPool *timed.JobPool
	lock            *sync.RWMutex
}

type IBaseBaseServiceManager interface {
	HasService(id string) bool
	GetService(id string) IBaseService
	RegisterService(string, IBaseService) error
	UnregisterService(string) error

	UnregisterAllServices() error

	MatchServiceByUri(uri string) IBaseService
	MatchServicesBy(condFunc func(IBaseService) bool) []IBaseService
	SupportsUri(uri string) bool

	CancelTimedJob(jobId int64) bool
	ScheduleTimeoutJob(job func()) int64

	AddUriRoute(service IBaseService, route string) (err error)
	RemoveUriRoute(route string) (success bool)
}

func NewBaseServiceManager(ctx *relay_common.WRContext) *BaseServiceManager {
	return &BaseServiceManager{
		trieTree:        uri.NewTrieTree(),
		serviceMap:      make(map[string]IBaseService),
		scheduleJobPool: ctx.TimedJobPool(),
		lock:            new(sync.RWMutex),
	}
}

func (m *BaseServiceManager) withWrite(cb func()) {
	m.lock.Lock()
	defer m.lock.RUnlock()
	cb()
}

func (s *BaseServiceManager) GetService(id string) IBaseService {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.serviceMap[id]
}

func (s *BaseServiceManager) HasService(id string) bool {
	return s.GetService(id) != nil
}

func (s *BaseServiceManager) RegisterService(clientId string, service IBaseService) error {
	return s.registerService(clientId, service)
}

// assume server has ensured client id exists
func (s *BaseServiceManager) registerService(clientId string, service IBaseService) error {
	s.withWrite(func() {
		s.serviceMap[service.Id()] = service
		for _, uri := range service.FullServiceUris() {
			s.trieTree.Add(uri, service, false)
		}
		service.OnStopped(func(service IBaseService) {
			for _, uri := range service.FullServiceUris() {
				s.trieTree.Remove(uri)
			}
		})
	})
	return nil
}

func (s *BaseServiceManager) UnregisterAllServices() error {
	errMsgBuilder := strings.Builder{}
	s.withWrite(func() {
		for _, service := range s.serviceMap {
			errMsgBuilder.WriteString(service.Stop().Error() + "\n")
			delete(s.serviceMap, service.Id())
		}
	})
	return errors.New(errMsgBuilder.String())
}

func (s *BaseServiceManager) UnregisterService(serviceId string) error {
	return s.unregisterService(serviceId)
}

func (s *BaseServiceManager) unregisterService(serviceId string) error {
	if !s.HasService(serviceId) {
		return errors.New("no such service " + serviceId)
	}
	s.withWrite(func() {
		s.serviceMap[serviceId].Stop()
		delete(s.serviceMap, serviceId)
	})
	return nil
}

func (s *BaseServiceManager) MatchServiceByUri(uri string) IBaseService {
	s.lock.RLock()
	defer s.lock.RUnlock()
	matchContext, err := s.trieTree.Match(uri)
	if matchContext == nil || err != nil {
		return nil
	}
	return matchContext.Value.(IBaseService)
}

func (s *BaseServiceManager) SupportsUri(uri string) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.trieTree.SupportsUri(uri)
}

func (s *BaseServiceManager) MatchServicesBy(condFunc func(IBaseService) bool) []IBaseService {
	s.lock.RLock()
	defer s.lock.RUnlock()
	var res []IBaseService
	for _, v := range s.serviceMap {
		if condFunc(v) {
			res = append(res, v)
		}
	}
	return res
}

func (s *BaseServiceManager) CancelTimedJob(jobId int64) bool {
	return s.scheduleJobPool.CancelJob(jobId)
}

func (s *BaseServiceManager) ScheduleTimeoutJob(job func()) int64 {
	return s.scheduleJobPool.ScheduleAsyncTimeoutJob(job, relay_server.ServiceKillTimeout)
}

func (s *BaseServiceManager) AddUriRoute(service IBaseService, route string) (err error) {
	s.withWrite(func() {
		err = s.trieTree.Add(route, service, true)
	})
	return err
}

func (s *BaseServiceManager) RemoveUriRoute(route string) (success bool) {
	s.withWrite(func() {
		success = s.trieTree.Remove(route)
	})
	return success
}
