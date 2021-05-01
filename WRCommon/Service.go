package WRCommon

import (
	"sync"
	"time"
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
	ServiceStatusRegistered   = 1
	ServiceStatusIdle         = 2
	ServiceStatusStarting     = 3
	ServiceStatusRunning      = 4
	ServiceStatusBlocked      = 5 // when queue is maxed out
	ServiceStatusDead         = 7 // health check fails
	ServiceStatusStopping     = 8
	ServiceStatusStopped      = 9 // then go back to idle
)

type Service struct {
	id                  string
	description         string
	owner               *WRBaseRole
	serviceUris         []string
	cTime               time.Time
	serviceType         int
	accessType          int
	executionType       int
	healthCheckInterval time.Duration
	status              int
	requestExecutor     IRequestExecutor
	lock                *sync.RWMutex
	pendingRequestQueue IBufferQueue
	processingPool      IServicePool
}

type IService interface {
	Id() string
	Description() string
	Owner() *WRBaseRole
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
	HealthCheck() bool
	Request(*Message) *Message // use trackable message to wait for the final state transition(Wait())
	Cancel(requestId string) error
	Kill(requestId string) error
	KillAllProcessingJobs() error
	CancelAllPendingJobs() error
	OnHealthCheckFails(cb func(IService))
	OnHealthRestored(cb func(service IService))
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

// TODO implementation
func (s *Service) Id() string {
	return s.id
}

func (s *Service) Description() string {
	return s.description
}

func (s *Service) Owner() *WRBaseRole {
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

func (s *Service) restartHealthCheckJob() {
	// TODO restart health check job
}

func (s *Service) SetHealthCheckInterval(duration time.Duration) {
	s.withWrite(func() {
		s.healthCheckInterval = duration
	})
	s.restartHealthCheckJob()
}

func (s *Service) Start() bool {
	// TODO start healthCheckJob, open a async pool for messages
	return false
}

func (s *Service) Stop() bool {
	// TODO stop healthCheckJob, close the async pool
	return false
}

func (s *Service) Status() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.status
}

func (s *Service) HealthCheck() bool {
	// TODO
	return false
}

func (s *Service) Request(*Message) *Message {
	// TODO use requestExecutor and do the job in the async pool
	return nil
}

func (s *Service) Cancel(requestId string) error {
	// TODO
	return nil
}

func (s *Service) Kill(requestId string) error {
	// TODO
	return nil
}

func (s *Service) KillAllProcessingJobs() error {
	// TODO
	return nil
}

func (s *Service) CancelAllPendingJobs() error {
	// TODO
	return nil
}

func (s *Service) OnHealthCheckFails(cb func(IService)) {
	// TODO
}

func (s *Service) OnHealthRestored(cb func(service IService)) {
	// TODO
}
