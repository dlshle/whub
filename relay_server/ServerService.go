package relay_server

import (
	"fmt"
	"strings"
	"sync"
	"time"
	"wsdk/relay_common"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/service"
)

// Service Uri should always be /service/serviceId/uri/params

// ServerService access type
const (
	ServiceAccessTypeHttp   = 0
	ServiceAccessTypeSocket = 1
	ServiceAccessTypeBoth   = 2
)

// ServerService execution type
const (
	ServiceExecutionAsync = 0
	ServiceExecutionSync  = 1
)

// ServerService type
const (
	ServiceTypeRelay = 0
)

const DefaultHealthCheckInterval = time.Minute * 30

type IServiceHost interface {
	relay_common.IDescribableRole
	RequestExecutor() relay_common.IRequestExecutor
	HealthCheckExecutor() relay_common.IHealthCheckExecutor
}

// TODO use service manager to mange services
type ServerService struct {
	uriPrefix           string
	ctx                 relay_common.IWRContext
	id                  string
	description         string
	host                IServiceHost
	serviceUris         []string
	cTime               time.Time
	serviceType         int
	accessType          int
	executionType       int
	healthCheckInterval time.Duration
	status              int
	healthCheckExecutor relay_common.IHealthCheckExecutor
	lock                *sync.RWMutex
	servicePool         service.IServicePool

	healthCheckJobId           int64
	healthCheckErrCallback     func(IServerService)
	healthCheckRestoreCallback func(IServerService)

	onStartedCallback func(IServerService)
	onStoppedCallback func(IServerService)
}

type IServerService interface {
	service.IBaseService
	Provider() IServiceHost
	HealthCheckInterval() time.Duration
	SetHealthCheckInterval(duration time.Duration)

	Register(IWRelayServer) error
	OnStarted(func(IServerService))
	OnStopped(func(IServerService))
	HealthCheck() error
	Request(*messages.Message) *messages.Message

	OnHealthCheckFails(cb func(IServerService))
	OnHealthRestored(cb func(service IServerService))

	RestoreExternally(reconnectedOwner *WRServerClient) error
}

func NewService(ctx relay_common.IWRContext, id string, description string, host IServiceHost, serviceUris []string, serviceType int, accessType int, exeType int) IServerService {
	return &ServerService{
		uriPrefix:           service.ServicePrefix + id,
		ctx:                 ctx,
		id:                  id,
		description:         description,
		host:                host,
		serviceUris:         serviceUris,
		cTime:               time.Now(),
		serviceType:         serviceType,
		accessType:          accessType,
		executionType:       exeType,
		healthCheckInterval: DefaultHealthCheckInterval,
		status:              service.ServiceStatusUnregistered,
		healthCheckExecutor: host.HealthCheckExecutor(),
		lock:                new(sync.RWMutex),
		servicePool:         service.NewServicePool(host.RequestExecutor(), service.MaxServicePoolSize),
		healthCheckJobId:    0,
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

func (s *ServerService) onHealthCheckFailedInternalHandler() {
	s.setStatus(service.ServiceStatusDead)
	if s.healthCheckErrCallback != nil {
		s.healthCheckErrCallback(s)
	}
}

func (s *ServerService) onHealthCheckRestoredInternalHandler() {
	s.setStatus(service.ServiceStatusRunning)
	s.reScheduleHealthCheckJob()
	if s.healthCheckRestoreCallback != nil {
		s.healthCheckRestoreCallback(s)
	}
}

func (s *ServerService) scheduleHealthCheckJob() int64 {
	onRetry := false
	s.withWrite(func() {
		s.healthCheckJobId = s.ctx.TimedJobPool().ScheduleAsyncIntervalJob(func() {
			err := s.HealthCheck()
			if err != nil {
				onRetry = true
				s.onHealthCheckFailedInternalHandler()
			} else if onRetry {
				// if err == nil && onRetry
				onRetry = false
				s.onHealthCheckRestoredInternalHandler()
			}
		}, s.healthCheckInterval)
	})
	return s.healthCheckJobId
}

func (s *ServerService) stopHealthCheckJob() {
	if s.healthCheckJobId != 0 {
		s.ctx.TimedJobPool().CancelJob(s.healthCheckJobId)
		s.withWrite(func() {
			s.healthCheckJobId = 0
		})
	}
}

func (s *ServerService) Id() string {
	return s.id
}

func (s *ServerService) Description() string {
	return s.description
}

func (s *ServerService) Provider() IServiceHost {
	return s.host
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

func (s *ServerService) HealthCheckInterval() time.Duration {
	return s.healthCheckInterval
}

func (s *ServerService) reScheduleHealthCheckJob() {
	if s.healthCheckJobId == 0 {
		s.scheduleHealthCheckJob()
		return
	}
	s.stopHealthCheckJob()
	s.scheduleHealthCheckJob()
}

func (s *ServerService) SetHealthCheckInterval(duration time.Duration) {
	s.withWrite(func() {
		s.healthCheckInterval = duration
	})
	s.reScheduleHealthCheckJob()
}

func (s *ServerService) Register(server IWRelayServer) (err error) {
	if err = server.RegisterService(s.Provider().Id(), s); err != nil {
		return
	}
	s.setStatus(service.ServiceStatusIdle)
	return
}

func (s *ServerService) Start() error {
	if s.Status() != service.ServiceStatusIdle {
		return NewInvalidServiceStatusTransitionError(s.Id(), s.Status(), service.ServiceStatusStarting)
	}
	s.scheduleHealthCheckJob()
	s.setStatus(service.ServiceStatusStarting)
	s.servicePool.Start()
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
	s.stopHealthCheckJob()
	s.setStatus(service.ServiceStatusStopping)
	s.servicePool.Stop()
	// after pool is stopped
	s.setStatus(service.ServiceStatusIdle)
	if s.onStoppedCallback != nil {
		s.onStoppedCallback(s)
	}
	return nil
}

func (s *ServerService) OnStarted(callback func(service IServerService)) {
	s.onStartedCallback = callback
}

func (s *ServerService) OnStopped(callback func(service IServerService)) {
	s.onStoppedCallback = callback
}

func (s *ServerService) RestoreExternally(reconnectedOwner *WRServerClient) (err error) {
	if s.Status() != service.ServiceStatusDead {
		err = NewInvalidServiceStatusError(s.Id(), s.Status(), fmt.Sprintf(" status should be %d to be restored externally", service.ServiceStatusDead))
		return
	}
	if err = s.Stop(); err != nil {
		return
	}
	oldOwner := s.host
	oldPool := s.servicePool
	oldHealthCheckExecutor := s.healthCheckExecutor
	s.withWrite(func() {
		s.host = reconnectedOwner
		s.servicePool = service.NewServicePool(reconnectedOwner.RequestExecutor(), s.servicePool.Size())
		s.healthCheckExecutor = reconnectedOwner.HealthCheckExecutor()
	})
	err = s.Start()
	if err != nil {
		// fallback to previous status
		s.withWrite(func() {
			s.host = oldOwner
			s.servicePool = oldPool
			s.healthCheckExecutor = oldHealthCheckExecutor
			s.status = service.ServiceStatusDead
		})
	}
	return err
}

func (s *ServerService) Status() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.status
}

func (s *ServerService) HealthCheck() (err error) {
	return s.healthCheckExecutor.DoHealthCheck()
}

func (s *ServerService) Request(message *messages.Message) *messages.Message {
	ServiceRequest := service.NewServiceRequest(message)
	s.servicePool.Add(ServiceRequest)
	if s.ExecutionType() == ServiceExecutionAsync {
		return nil
	} else {
		return ServiceRequest.Response()
	}
}

func (s *ServerService) Cancel(messageId string) error {
	return s.servicePool.Cancel(messageId)
}

func (s *ServerService) KillAllProcessingJobs() error {
	return s.servicePool.KillAll()
}

func (s *ServerService) CancelAllPendingJobs() error {
	return s.servicePool.CancelAll()
}

func (s *ServerService) OnHealthCheckFails(cb func(IServerService)) {
	s.withWrite(func() {
		s.healthCheckErrCallback = cb
	})
}

func (s *ServerService) OnHealthRestored(cb func(service IServerService)) {
	s.withWrite(func() {
		s.healthCheckRestoreCallback = cb
	})
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
