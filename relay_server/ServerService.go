package relay_server

import (
	"fmt"
	"sync"
	"time"
	"wsdk/relay_common"
	"wsdk/relay_common/messages"
)

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

// ServerService status
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

type ServerService struct {
	ctx                 relay_common.IWRContext
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
	healthCheckExecutor relay_common.IHealthCheckExecutor
	lock                *sync.RWMutex
	servicePool         IServicePool

	healthCheckJobId    int64
	healthCheckErrCallback func(IServerService)
	healthCheckRestoreCallback func(IServerService)
}

type IServerService interface {
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

	Register(IWRelayServer) error
	Start() error
	Stop() error
	Status() int
	HealthCheck() error
	Request(*messages.Message) *messages.Message
	Cancel(messageId string) error
	KillAllProcessingJobs() error
	CancelAllPendingJobs() error
	OnHealthCheckFails(cb func(IServerService))
	OnHealthRestored(cb func(service IServerService))

	RestoreExternally(reconnectedOwner IWRServerClient) error

	Describe() relay_common.ServiceDescriptor
}

func NewService(ctx relay_common.IWRContext, id string, description string, owner IWRServerClient, serviceUris []string, serviceType int, accessType int, exeType int) IServerService {
	return &ServerService{
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
	s.setStatus(ServiceStatusDead)
	if s.healthCheckErrCallback != nil {
		s.healthCheckErrCallback(s)
	}
}

func (s *ServerService) onHealthCheckRestoredInternalHandler() {
	s.setStatus(ServiceStatusRunning)
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

func (s *ServerService) Owner() IWRServerClient {
	return s.owner
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
	if err = server.RegisterService(s); err != nil {
		return
	}
	s.setStatus(ServiceStatusIdle)
	return
}

func (s *ServerService) Start() error {
	if s.Status() != ServiceStatusIdle {
		return NewInvalidServiceStatusTransitionError(s.Id(), s.Status(), ServiceStatusStarting)
	}
	s.scheduleHealthCheckJob()
	s.setStatus(ServiceStatusStarting)
	s.servicePool.Start()
	s.setStatus(ServiceStatusRunning)
	return nil
}

func (s *ServerService) Stop() error {
	if s.Status() > ServiceStatusIdle || s.Status() < ServiceStatusStopping {
		return NewInvalidServiceStatusTransitionError(s.Id(), s.Status(), ServiceStatusStopping)
	}
	s.stopHealthCheckJob()
	s.setStatus(ServiceStatusStopping)
	s.servicePool.Stop()
	// after pool is stopped
	s.setStatus(ServiceStatusIdle)
	return nil
}

func (s *ServerService) RestoreExternally(reconnectedOwner IWRServerClient) (err error) {
	if s.Status() != ServiceStatusDead {
		err = NewInvalidServiceStatusError(s.Id(), s.Status(), fmt.Sprintf(" status should be %d to be restored externally", ServiceStatusDead))
		return
	}
	if err = s.Stop(); err != nil {
		return
	}
	oldOwner := s.owner
	oldPool := s.servicePool
	oldHealthCheckExecutor := s.healthCheckExecutor
	s.withWrite(func() {
		s.owner = reconnectedOwner
		s.servicePool = NewServicePool(reconnectedOwner.RequestExecutor(), s.servicePool.Size())
		s.healthCheckExecutor = reconnectedOwner.HealthCheckExecutor()
	})
	err = s.Start()
	if err != nil {
		// fallback to previous status
		s.withWrite(func() {
			s.owner = oldOwner
			s.servicePool = oldPool
			s.healthCheckExecutor = oldHealthCheckExecutor
			s.status = ServiceStatusDead
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
	serviceMessage := messages.NewServiceMessage(message)
	s.servicePool.Add(serviceMessage)
	if s.ExecutionType() == ServiceExecutionAsync {
		return nil
	} else {
		return serviceMessage.Response()
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

func (s *ServerService) Describe() relay_common.ServiceDescriptor {
	return relay_common.ServiceDescriptor{
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
