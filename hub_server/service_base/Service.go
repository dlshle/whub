package service_base

import (
	"fmt"
	"strings"
	"sync"
	"time"
	"whub/common/logger"
	"whub/hub_common/messages"
	"whub/hub_common/roles"
	"whub/hub_common/service"
	"whub/hub_server/context"
	"whub/hub_server/module_base"
	"whub/hub_server/modules/metering"
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
	logger        *logger.SimpleLogger
	metering      metering.IMeteringModule `module:""`

	onStartedCallback func(baseService service.IBaseService)
	onStoppedCallback func(baseService service.IBaseService)
}

// TODO need a safe status transitioning method!
type IService interface {
	service.IBaseService
	Provider() IServiceProvider
	Kill() error
	UriPrefix() string
	Logger() *logger.SimpleLogger
}

func NewService(id string, description string, provider IServiceProvider, executor service.IRequestExecutor, serviceUris []string, serviceType int, accessType int, exeType int) *Service {
	service := &Service{
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
		serviceQueue:  service.NewServiceTaskQueue(context.Ctx.Server().Id(), executor, context.Ctx.ServiceTaskPool()),
		logger:        context.Ctx.Logger().WithPrefix(fmt.Sprintf("[Service-%s]", id)),
	}
	err := module_base.Manager.AutoFill(service)
	if err != nil {
		panic(err)
	}
	return service
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
		s.logger.Printf("service can not be started with current status [%s]", service.ServiceStatusStringMap[s.Status()])
		return NewInvalidServiceStatusTransitionError(s.Id(), s.Status(), service.ServiceStatusStarting)
	}
	s.setStatus(service.ServiceStatusStarting)
	s.logger.Println("service is starting")
	// some starting task maybe
	s.setStatus(service.ServiceStatusRunning)
	s.logger.Println("service is running")
	if s.onStartedCallback != nil {
		s.onStartedCallback(s)
	}
	return nil
}

func (s *Service) Stop() error {
	if !(s.Status() > service.ServiceStatusIdle || s.Status() < service.ServiceStatusStopping) {
		s.logger.Printf("service can not be stopped with current status [%s]", service.ServiceStatusStringMap[s.Status()])
		return NewInvalidServiceStatusTransitionError(s.Id(), s.Status(), service.ServiceStatusStopping)
	}
	s.setStatus(service.ServiceStatusStopping)
	s.logger.Println("service is stopping")
	// set executor to nil
	s.serviceQueue.Stop()
	s.setStatus(service.ServiceStatusIdle)
	s.logger.Println("service has stopped, current status is ", service.ServiceStatusStringMap[s.Status()])
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

func (s *Service) Handle(request service.IServiceRequest) messages.IMessage {
	s.logger.Println("handle request ", request.String())
	s.serviceQueue.Schedule(request)
	s.traceMessagePerformance(request.Id(), "request in queue")
	if s.ExecutionType() == service.ServiceExecutionAsync {
		return nil
	} else {
		resp := request.Response()
		s.traceMessagePerformance(request.Id(), "sync request handled")
		return resp
	}
}

func (s *Service) traceMessagePerformance(id, description string) {
	s.metering.Track(s.metering.GetAssembledTraceId(metering.TMessagePerformance, id), description)
}

func (s *Service) Cancel(messageId string) error {
	s.logger.Println("cancel request ", messageId)
	return s.serviceQueue.Cancel(messageId)
}

func (s *Service) KillAllProcessingJobs() error {
	s.logger.Println("kill all processing jobs")
	return s.serviceQueue.KillAll()
}

func (s *Service) CancelAllPendingJobs() error {
	s.logger.Println("cancel all pending jobs")
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
	s.logger.Println("killing service...")
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

func (s *Service) Logger() *logger.SimpleLogger {
	return s.logger
}
