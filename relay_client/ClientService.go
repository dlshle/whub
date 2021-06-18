package relay_client

import (
	"errors"
	"strings"
	"sync"
	"time"
	"wsdk/relay_common"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/service"
)

type ClientService struct {
	ctx *relay_common.WRContext

	serviceCenterClient service.IServiceCenterClient
	servicePool         service.IServicePool

	id          string
	uriPrefix   string
	description string
	serviceUris []string
	// requestHandlers map[string]messages.MessageHandlerFunc
	handler       service.IServiceHandler
	hostInfo      *relay_common.RoleDescriptor
	serviceType   int
	accessType    int
	executionType int
	descriptor    *service.ServiceDescriptor
	cTime         time.Time

	status             int
	healthCheckHandler *service.ServiceHealthCheckHandler

	onHealthCheckFailsCallback    func(service IClientService)
	onHealthCheckRestoredCallback func(service IClientService)

	lock *sync.RWMutex
}

// TODO NewFunc
func NewClientService(ctx *relay_common.WRContext, id string, server *relay_common.WRServer) *ClientService {
	handler := service.NewServiceHandler()
	return &ClientService{
		id:                  id,
		ctx:                 ctx,
		serviceCenterClient: service.NewServiceCenterClient(ctx, server),
		handler:             handler,
		servicePool:         service.NewServicePool(NewClientServiceExecutor(ctx, handler), service.MaxServicePoolSize/2),
	}
}

type IClientService interface {
	service.IBaseService
	UpdateDescription(string) error
	Handle(*messages.Message) error
	HostInfo() relay_common.RoleDescriptor
	RegisterRoute(shortUri string, handler service.RequestHandler) error // should update service descriptor to the host
	UnregisterRoute(shortUri string) error                               // should update service descriptor to the host
	NotifyHostForUpdate() error
	NewMessage(to string, uri string, msgType int, payload []byte) *messages.Message

	Register() error

	HealthCheck() error
	OnHealthCheckFails(cb func(service IClientService))
	OnHealthRestored(cb func(service IClientService))
}

func (s *ClientService) withWrite(cb func()) {
	s.lock.Lock()
	defer s.lock.Unlock()
	cb()
}

func (s *ClientService) Id() string {
	return s.id
}

func (s *ClientService) Description() string {
	return s.description
}

func (s *ClientService) ServiceUris() []string {
	return s.serviceUris
}

func (s *ClientService) FullServiceUris() []string {
	fullUris := make([]string, len(s.serviceUris))
	for i, uri := range s.ServiceUris() {
		fullUris[i] = s.uriPrefix + uri
	}
	return fullUris
}

func (s *ClientService) SupportsUri(uri string) bool {
	if !strings.HasPrefix(uri, s.uriPrefix) {
		return false
	}
	actualUri := strings.TrimPrefix(uri, s.uriPrefix)
	for _, v := range s.ServiceUris() {
		if strings.HasPrefix(actualUri, v) {
			return true
		}
	}
	return false
}

func (s *ClientService) CTime() time.Time {
	return s.cTime
}

func (s *ClientService) UpdateDescription(desc string) (err error) {
	err = s.NotifyHostForUpdate()
	if err == nil {
		s.withWrite(func() {
			s.description = desc
		})
	}
	return
}

func (s *ClientService) ServiceType() int {
	return s.serviceType
}

func (s *ClientService) AccessType() int {
	return s.accessType
}

func (s *ClientService) ExecutionType() int {
	return s.executionType
}

func (s *ClientService) HostInfo() relay_common.RoleDescriptor {
	return *s.hostInfo
}

// returns the corresponding raw uri of the service or ""
func (s *ClientService) matchUri(uri string) (string, error) {
	actualUri := strings.TrimPrefix(uri, s.uriPrefix)
	for _, v := range s.ServiceUris() {
		if strings.HasPrefix(actualUri, v) {
			return v, nil
		}
	}
	return "", errors.New("no matched uri")
}

func (s *ClientService) Handle(message *messages.Message) error {
	request := service.NewServiceRequest(message)
	return s.handler.Handle(request)
}

func (s *ClientService) RegisterRoute(shortUri string, handler service.RequestHandler) error {
	s.withWrite(func() {
		s.serviceUris = append(s.serviceUris, shortUri)
		s.handler.Register(shortUri, handler)
	})
	return s.NotifyHostForUpdate()
}

func (s *ClientService) UnregisterRoute(shortUri string) error {
	uriIndex := -1
	for i, uri := range s.ServiceUris() {
		if uri == shortUri {
			uriIndex = i
		}
	}
	if uriIndex == -1 {
		return errors.New("shortUri " + shortUri + " does not exist")
	}
	s.withWrite(func() {
		l := len(s.serviceUris)
		s.serviceUris[l-1], s.serviceUris[uriIndex] = s.serviceUris[uriIndex], s.serviceUris[l-1]
		s.serviceUris = s.serviceUris[:l-1]
		s.handler.Unregister(shortUri)
	})
	return s.NotifyHostForUpdate()
}

func (s *ClientService) NotifyHostForUpdate() error {
	if s.serviceCenterClient != nil {
		return s.serviceCenterClient.UpdateService(s.Describe())
	}
	return errors.New("no serviceCenterClient found")
}

func (s *ClientService) NewMessage(to string, uri string, msgType int, payload []byte) *messages.Message {
	return s.ctx.Identity().DraftMessage(s.ctx.Identity().Id(), to, uri, msgType, payload)
}

func (s *ClientService) Describe() service.ServiceDescriptor {
	return service.ServiceDescriptor{
		Id:            s.Id(),
		Description:   s.Description(),
		HostInfo:      s.HostInfo(),
		Provider:      s.ctx.Identity().Describe(),
		ServiceUris:   s.ServiceUris(),
		CTime:         s.CTime(),
		ServiceType:   s.ServiceType(),
		AccessType:    s.AccessType(),
		ExecutionType: s.ExecutionType(),
		Status:        s.Status(),
	}
}

func (s *ClientService) Status() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.status
}

func (s *ClientService) Register() (err error) {
	err = s.serviceCenterClient.RegisterService(s.Describe())
	if err == nil {
		s.withWrite(func() {
			s.status = service.ServiceStatusRegistered
		})
	}
	return
}

func (s *ClientService) Start() (err error) {
	if s.Status() != service.ServiceStatusRegistered {
		return errors.New("invalid status to start a service(status != ServiceStatusRegistered)")
	}
	s.withWrite(func() {
		s.status = service.ServiceStatusStarting
	})
	s.healthCheckHandler.StartHealthCheck()
	s.withWrite(func() {
		s.status = service.ServiceStatusRunning
	})
	// Notify service started
	return
}

func (s *ClientService) Stop() error {
	if !(s.Status() > service.ServiceStatusUnregistered && s.Status() < service.ServiceStatusStopping) {
		return errors.New("invalid status to stop a service")
	}
	s.healthCheckHandler.StopHealthCheck()
	err := s.unregister()
	if err != nil {
		return err
	}
	s.withWrite(func() {
		s.status = service.ServiceStatusStopping
	})
	s.servicePool.Stop()
	// after pool is stopped
	s.withWrite(func() {
		s.status = service.ServiceStatusUnregistered
	})
	/*
		if s.onStoppedCallback != nil {
			s.onStoppedCallback(s)
		}
	*/
	return nil
}

func (s *ClientService) unregister() error {
	return s.serviceCenterClient.UnregisterService(s.Describe())
}

func (s *ClientService) HealthCheck() error {
	return s.serviceCenterClient.HealthCheck()
}

func (s *ClientService) OnHealthCheckFails(cb func(service IClientService)) {
	s.withWrite(func() {
		s.onHealthCheckFailsCallback = cb
	})
}

func (s *ClientService) OnHealthRestored(cb func(service IClientService)) {
	s.withWrite(func() {
		s.onHealthCheckRestoredCallback = cb
	})
}

func (s *ClientService) onHealthCheckFailedInternalHandler() {
	s.withWrite(func() {
		s.status = service.ServiceStatusDead
	})
	if s.onHealthCheckFailsCallback != nil {
		s.onHealthCheckRestoredCallback(s)
	}
}

func (s *ClientService) onHealthCheckRestoredInternalHandler() {
	s.withWrite(func() {
		s.status = service.ServiceStatusRunning
	})
	if s.onHealthCheckFailsCallback != nil {
		s.onHealthCheckRestoredCallback(s)
	}
}

func (s *ClientService) Cancel(messageId string) error {
	return s.servicePool.Cancel(messageId)
}

func (s *ClientService) KillAllProcessingJobs() error {
	return s.servicePool.KillAll()
}

func (s *ClientService) CancelAllPendingJobs() error {
	return s.servicePool.CancelAll()
}
