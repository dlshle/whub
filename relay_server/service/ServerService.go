package service

import (
	"fmt"
	"strings"
	"sync"
	"time"
	"wsdk/relay_common"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/service"
)

type IServiceProvider = relay_common.IDescribableRole

type ServerService struct {
	uriPrefix     string
	ctx           relay_common.IWRContext
	id            string
	description   string
	provider      IServiceProvider
	serviceUris   []string
	cTime         time.Time
	serviceType   int
	accessType    int
	executionType int
	status        int
	lock          *sync.RWMutex
	serviceQueue  service.IServiceTaskQueue

	onStartedCallback func(baseService service.IBaseService)
	onStoppedCallback func(baseService service.IBaseService)
}

// TODO need a safe status transitioning method!
type IServerService interface {
	service.IBaseService
	Provider() IServiceProvider
	Kill() error
}

func NewService(ctx relay_common.IWRContext, id string, description string, provider IServiceProvider, executor relay_common.IRequestExecutor, serviceUris []string, serviceType int, accessType int, exeType int) *ServerService {
	return &ServerService{
		uriPrefix:     fmt.Sprintf("%s/%s", service.ServicePrefix, id),
		ctx:           ctx,
		id:            id,
		description:   description,
		provider:      provider,
		serviceUris:   serviceUris,
		cTime:         time.Now(),
		serviceType:   serviceType,
		accessType:    accessType,
		executionType: exeType,
		status:        service.ServiceStatusIdle,
		lock:          new(sync.RWMutex),
		serviceQueue:  service.NewServiceTaskQueue(executor, ctx.ServiceTaskPool()),
	}
}

func (s *ServerService) withWrite(cb func()) {
	s.lock.Lock()
	defer s.lock.Unlock()
	cb()
}

func (s *ServerService) setStatus(status int) {
	s.withWrite(func() {
		s.status = status
	})
}

func (s *ServerService) Id() string {
	return s.id
}

func (s *ServerService) Description() string {
	return s.description
}

func (s *ServerService) Provider() IServiceProvider {
	return s.provider
}

func (s *ServerService) ServiceType() int {
	return s.serviceType
}

func (s *ServerService) ServiceUris() []string {
	return s.serviceUris
}

func (s *ServerService) CreationTime() time.Time {
	return s.cTime
}

func (s *ServerService) AccessType() int {
	return s.accessType
}

func (s *ServerService) ExecutionType() int {
	return s.executionType
}

func (s *ServerService) CTime() time.Time {
	return s.cTime
}

func (s *ServerService) Start() error {
	if s.Status() != service.ServiceStatusIdle {
		return NewInvalidServiceStatusTransitionError(s.Id(), s.Status(), service.ServiceStatusStarting)
	}
	s.setStatus(service.ServiceStatusStarting)
	s.serviceQueue.Start()
	s.setStatus(service.ServiceStatusRunning)
	if s.onStartedCallback != nil {
		s.onStartedCallback(s)
	}
	return nil
}

func (s *ServerService) Stop() error {
	if !(s.Status() > service.ServiceStatusIdle || s.Status() < service.ServiceStatusStopping) {
		return NewInvalidServiceStatusTransitionError(s.Id(), s.Status(), service.ServiceStatusStopping)
	}
	s.setStatus(service.ServiceStatusStopping)
	s.serviceQueue.Stop()
	// after pool is stopped
	s.setStatus(service.ServiceStatusIdle)
	if s.onStoppedCallback != nil {
		s.onStoppedCallback(s)
	}
	return nil
}

func (s *ServerService) OnStarted(callback func(service service.IBaseService)) {
	s.onStartedCallback = callback
}

func (s *ServerService) OnStopped(callback func(service service.IBaseService)) {
	s.onStoppedCallback = callback
}

func (s *ServerService) Status() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.status
}

func (s *ServerService) Handle(message *messages.Message) *messages.Message {
	serviceRequest := service.NewServiceRequest(message)
	s.serviceQueue.Add(serviceRequest)
	if s.ExecutionType() == service.ServiceExecutionAsync {
		return nil
	} else {
		return serviceRequest.Response()
	}
}

func (s *ServerService) Cancel(messageId string) error {
	return s.serviceQueue.Cancel(messageId)
}

func (s *ServerService) KillAllProcessingJobs() error {
	return s.serviceQueue.KillAll()
}

func (s *ServerService) CancelAllPendingJobs() error {
	return s.serviceQueue.CancelAll()
}

func (s *ServerService) Describe() service.ServiceDescriptor {
	return service.ServiceDescriptor{
		Id:            s.Id(),
		Description:   s.Description(),
		HostInfo:      s.ctx.Identity().Describe(),
		Provider:      s.Provider().Describe(),
		ServiceUris:   s.ServiceUris(),
		CTime:         s.CreationTime(),
		ServiceType:   s.ServiceType(),
		AccessType:    s.AccessType(),
		ExecutionType: s.ExecutionType(),
		Status:        s.Status(),
	}
}

func (s *ServerService) SupportsUri(uri string) bool {
	if !strings.HasPrefix(uri, s.uriPrefix) {
		return false
	}
	actualUri := strings.TrimPrefix(uri, s.uriPrefix)
	for _, uri := range s.ServiceUris() {
		if strings.HasPrefix(actualUri, uri) {
			return true
		}
	}
	return false
}

func (s *ServerService) FullServiceUris() []string {
	fullUris := make([]string, len(s.serviceUris))
	for i, uri := range s.ServiceUris() {
		fullUris[i] = s.uriPrefix + uri
	}
	return fullUris
}

func (s *ServerService) Kill() error {
	s.setStatus(service.ServiceStatusDead)
	return s.KillAllProcessingJobs()
}

func (s *ServerService) ProviderInfo() relay_common.RoleDescriptor {
	return s.Provider().Describe()
}

func (s *ServerService) HostInfo() relay_common.RoleDescriptor {
	return s.ctx.Identity().Describe()
}
