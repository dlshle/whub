package service

import (
	"fmt"
	"strings"
	"sync"
	"time"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/roles"
	"wsdk/relay_common/service"
	"wsdk/relay_server/context"
)

const (
	MaxServicePerClient = 5
	ServiceKillTimeout  = time.Minute * 15
)

type IServiceProvider = roles.IDescribableRole

type Service struct {
	uriPrefix     string
	ctx           *context.Context
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
type IService interface {
	service.IBaseService
	Provider() IServiceProvider
	Kill() error
	UriPrefix() string
}

func NewService(id string, description string, provider IServiceProvider, executor service.IRequestExecutor, serviceUris []string, serviceType int, accessType int, exeType int) *Service {
	return &Service{
		uriPrefix:     fmt.Sprintf("%s/%s", service.ServicePrefix, id),
		ctx:           context.Ctx,
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
		serviceQueue:  service.NewServiceTaskQueue(executor, context.Ctx.ServiceTaskPool()),
	}
}

func (s *Service) withWrite(cb func()) {
	s.lock.Lock()
	defer s.lock.Unlock()
	cb()
}

func (s *Service) setStatus(status int) {
	s.withWrite(func() {
		s.status = status
	})
}

func (s *Service) Id() string {
	return s.id
}

func (s *Service) Description() string {
	return s.description
}

func (s *Service) Provider() IServiceProvider {
	return s.provider
}

func (s *Service) ServiceType() int {
	return s.serviceType
}

func (s *Service) ServiceUris() []string {
	return s.serviceUris
}

func (s *Service) CreationTime() time.Time {
	return s.cTime
}

func (s *Service) AccessType() int {
	return s.accessType
}

func (s *Service) ExecutionType() int {
	return s.executionType
}

func (s *Service) CTime() time.Time {
	return s.cTime
}

func (s *Service) Start() error {
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

func (s *Service) Stop() error {
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

func (s *Service) OnStarted(callback func(service service.IBaseService)) {
	s.onStartedCallback = callback
}

func (s *Service) OnStopped(callback func(service service.IBaseService)) {
	s.onStoppedCallback = callback
}

func (s *Service) Status() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.status
}

func (s *Service) Handle(message *messages.Message) *messages.Message {
	if strings.HasPrefix(message.Uri(), s.uriPrefix) {
		message = message.SetUri(strings.TrimPrefix(message.Uri(), s.uriPrefix))
	}
	serviceRequest := service.NewServiceRequest(message)
	s.serviceQueue.Schedule(serviceRequest)
	if s.ExecutionType() == service.ServiceExecutionAsync {
		return nil
	} else {
		return serviceRequest.Response()
	}
}

func (s *Service) Cancel(messageId string) error {
	return s.serviceQueue.Cancel(messageId)
}

func (s *Service) KillAllProcessingJobs() error {
	return s.serviceQueue.KillAll()
}

func (s *Service) CancelAllPendingJobs() error {
	return s.serviceQueue.CancelAll()
}

func (s *Service) Describe() service.ServiceDescriptor {
	return service.ServiceDescriptor{
		Id:            s.Id(),
		Description:   s.Description(),
		HostInfo:      s.ctx.Server().Describe(),
		Provider:      s.Provider().Describe(),
		ServiceUris:   s.ServiceUris(),
		CTime:         s.CreationTime(),
		ServiceType:   s.ServiceType(),
		AccessType:    s.AccessType(),
		ExecutionType: s.ExecutionType(),
		Status:        s.Status(),
	}
}

func (s *Service) SupportsUri(uri string) bool {
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

func (s *Service) FullServiceUris() []string {
	fullUris := make([]string, len(s.serviceUris))
	for i, uri := range s.ServiceUris() {
		fullUris[i] = s.uriPrefix + uri
	}
	return fullUris
}

func (s *Service) Kill() error {
	s.setStatus(service.ServiceStatusDead)
	return s.KillAllProcessingJobs()
}

func (s *Service) ProviderInfo() roles.RoleDescriptor {
	return s.Provider().Describe()
}

func (s *Service) HostInfo() roles.RoleDescriptor {
	return s.ctx.Server().Describe()
}

func (s *Service) UriPrefix() string {
	return s.uriPrefix
}