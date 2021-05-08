package WRelayServer

import (
	"sync"
	"time"
	"wsdk/WRCommon"
	"wsdk/WRCommon/Message"
)

// Service access type
const (
	ServiceAccessTypeHttp   = 0
	ServiceAccessTypeSocket = 1
	ServiceAccessTypeBoth   = 2
)

// Service execution type
const (
	ServiceExecutionAsync = 0
	ServiceExecutionSync  = 1
)

// Service type
const (
	ServiceTypeRelay = 0
)

// Service status
const (
	ServiceStatusUnregistered = 0
	ServiceStatusIdle         = 1
	ServiceStatusStarting     = 2
	ServiceStatusRunning      = 3
	ServiceStatusBlocked      = 4 // when queue is maxed out
	ServiceStatusDead         = 5 // health check fails
	ServiceStatusStopping     = 6
)

const DefaultHealthCheckInterval = time.Minute * 30

type Service struct {
	ctx					WRCommon.IWRContext
	id                  string
	description         string
	owner               IWRServerClient
	serviceUris         []string
	cTime               time.Time
	serviceType         int
	accessType          int
	executionType       int
	healthCheckInterval time.Duration
	status              int
	healthCheckExecutor WRCommon.IHealthCheckExecutor
	lock                *sync.RWMutex
	servicePool         IServicePool

	healthCheckJobId    int64
	healthCheckErrCallback func(IService)
	healthCheckRestoreCallback func(IService)
}

type IService interface {
	Id() string
	Description() string
	Owner() IWRServerClient
	ServiceType() int
	ServiceUris() []string
	CreationTime() time.Time
	AccessType() int
	ExecutionType() int
	HealthCheckInterval() time.Duration
	SetHealthCheckInterval(duration time.Duration)

	Start() bool
	Stop() bool
	Status() int
	HealthCheck() error
	Request(*Message.Message) *Message.Message
	Cancel(messageId string) error
	KillAllProcessingJobs() error
	CancelAllPendingJobs() error
	OnHealthCheckFails(cb func(IService))
	OnHealthRestored(cb func(service IService))

	Describe() WRCommon.ServiceDescriptor
}

func NewService(ctx WRCommon.IWRContext, id string, description string, owner IWRServerClient, serviceUris []string, serviceType int, accessType int, exeType int) IService {
	return &Service{
		ctx: ctx,
		id: id,
		description: description,
		owner: owner,
		serviceUris: serviceUris,
		cTime: time.Now(),
		serviceType: serviceType,
		accessType: accessType,
		executionType: exeType,
		healthCheckInterval: DefaultHealthCheckInterval,
		status: ServiceStatusUnregistered,
		healthCheckExecutor: owner.HealthCheckExecutor(),
		lock: new(sync.RWMutex),
		servicePool: NewServicePool(owner.RequestExecutor(), MaxServicePoolSize),
		healthCheckJobId: 0,
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

func (s *Service) onHealthCheckFailedInternalHandler() {
	s.setStatus(ServiceStatusDead)
	if s.healthCheckErrCallback != nil {
		s.healthCheckErrCallback(s)
	}
}

func (s *Service) onHealthCheckRestoredInternalHandler() {
	s.setStatus(ServiceStatusRunning)
	if s.healthCheckRestoreCallback != nil {
		s.healthCheckRestoreCallback(s)
	}
}

func (s *Service) scheduleHealthCheckJob() int64 {
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

func (s *Service) stopHealthCheckJob() {
	if s.healthCheckJobId != 0 {
		s.ctx.TimedJobPool().CancelJob(s.healthCheckJobId)
		s.withWrite(func() {
			s.healthCheckJobId = 0
		})
	}
}

func (s *Service) Id() string {
	return s.id
}

func (s *Service) Description() string {
	return s.description
}

func (s *Service) Owner() IWRServerClient {
	return s.owner
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

func (s *Service) HealthCheckInterval() time.Duration {
	return s.healthCheckInterval
}

func (s *Service) reScheduleHealthCheckJob() {
	if s.healthCheckJobId == 0 {
		s.scheduleHealthCheckJob()
		return
	}
	s.stopHealthCheckJob()
	s.scheduleHealthCheckJob()
}

func (s *Service) SetHealthCheckInterval(duration time.Duration) {
	s.withWrite(func() {
		s.healthCheckInterval = duration
	})
	s.reScheduleHealthCheckJob()
}

func (s *Service) Start() bool {
	if s.Status() != ServiceStatusIdle {
		return false
	}
	s.scheduleHealthCheckJob()
	s.setStatus(ServiceStatusStarting)
	s.servicePool.Start()
	s.setStatus(ServiceStatusRunning)
	return false
}

func (s *Service) Stop() bool {
	if s.Status() > ServiceStatusIdle || s.Status() < ServiceStatusStopping {
		return false
	}
	s.stopHealthCheckJob()
	s.setStatus(ServiceStatusStopping)
	s.servicePool.Stop()
	// after pool is stopped
	s.setStatus(ServiceStatusIdle)
	return false
}

func (s *Service) Status() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.status
}

func (s *Service) HealthCheck() (err error) {
	return s.healthCheckExecutor.DoHealthCheck()
}

func (s *Service) Request(message *Message.Message) *Message.Message {
	serviceMessage := Message.NewServiceMessage(message)
	s.servicePool.Add(serviceMessage)
	if s.ExecutionType() == ServiceExecutionAsync {
		return nil
	} else {
		return serviceMessage.Response()
	}
}

func (s *Service) Cancel(messageId string) error {
	return s.servicePool.Cancel(messageId)
}

func (s *Service) KillAllProcessingJobs() error {
	return s.servicePool.KillAll()
}

func (s *Service) CancelAllPendingJobs() error {
	return s.servicePool.CancelAll()
}

func (s *Service) OnHealthCheckFails(cb func(IService)) {
	s.withWrite(func() {
		s.healthCheckErrCallback = cb
	})
}

func (s *Service) OnHealthRestored(cb func(service IService)) {
	s.withWrite(func() {
		s.healthCheckRestoreCallback = cb
	})
}

func (s *Service) Describe() WRCommon.ServiceDescriptor {
	return WRCommon.ServiceDescriptor{
		Id:            s.Id(),
		Description:   s.Description(),
		HostInfo:      s.ctx.Identity().Describe(),
		Owner:         s.Owner(),
		ServiceUris:   s.ServiceUris(),
		CTime:         s.CreationTime(),
		ServiceType:   s.ServiceType(),
		AccessType:    s.AccessType(),
		ExecutionType: s.ExecutionType(),
	}
}
