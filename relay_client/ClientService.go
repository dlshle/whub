package relay_client

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"wsdk/common/logger"
	"wsdk/relay_client/container"
	"wsdk/relay_client/context"
	"wsdk/relay_client/controllers"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/health_check"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/roles"
	"wsdk/relay_common/service"
)

type ClientService struct {
	ctx context.IContext

	serviceManagerClient IServiceManagerClient
	serviceTaskQueue     service.IServiceTaskQueue

	id            string
	uriPrefix     string
	description   string
	serviceUris   []string // shortUris
	handler       service.IServiceHandler
	host          roles.ICommonServer
	serviceType   int
	accessType    int
	executionType int
	cTime         time.Time

	status int

	onStartedCallback func(service.IBaseService)
	onStoppedCallback func(service.IBaseService)

	healthCheckHandler            *health_check.HealthCheckHandler
	onHealthCheckFailsCallback    func(service IClientService)
	onHealthCheckRestoredCallback func(service IClientService)

	m controllers.IClientMeteringController `$inject:""`

	lock *sync.RWMutex

	logger *logger.SimpleLogger
}

func NewClientService(id string, description string, accessType int, execType int, server roles.ICommonServer, serverConn connection.IConnection) *ClientService {
	handler := service.NewServiceHandler()
	s := &ClientService{
		id:                   id,
		description:          description,
		ctx:                  context.Ctx,
		serviceManagerClient: NewServiceCenterClient(context.Ctx.Identity().Id(), server.Id(), serverConn),
		serviceTaskQueue:     service.NewServiceTaskQueue(context.Ctx.Identity().Id(), NewClientServiceExecutor(handler), context.Ctx.ServiceTaskPool()),
		handler:              handler,
		host:                 server,
		lock:                 new(sync.RWMutex),
		uriPrefix:            fmt.Sprintf("%s/%s", service.ServicePrefix, id),
		logger:               context.Ctx.Logger().WithPrefix(fmt.Sprintf("[%s]", id)),
		serviceType:          service.ServiceTypeRelay,
		accessType:           accessType,
		executionType:        execType,
	}
	s.init()
	err := container.Container.Fill(s)
	if err != nil {
		panic(err)
	}
	return s
}

type IClientService interface {
	service.IBaseService
	Init(server roles.ICommonServer, serverConn connection.IConnection) error
	UpdateDescription(string) error
	RegisterRoute(shortUri string, handler service.RequestHandler) error // should update service descriptor to the host
	UnregisterRoute(shortUri string) error                               // should update service descriptor to the host
	NotifyHostForUpdate() error
	NewMessage(to string, uri string, msgType int, payload []byte) *messages.Message

	Register() error

	HealthCheck() error
	OnHealthCheckFails(cb func(service IClientService))
	OnHealthRestored(cb func(service IClientService))

	ResolveByAck(request *service.ServiceRequest) error
	ResolveByResponse(request *service.ServiceRequest, responseData []byte) error

	Logger() *logger.SimpleLogger
}

func (s *ClientService) init() {
	s.status = service.ServiceStatusUnregistered
	s.healthCheckHandler = health_check.NewHealthCheckHandler(
		health_check.DefaultHealthCheckInterval,
		s.HealthCheck,
		s.onHealthCheckFailedInternalHandler,
		s.onHealthCheckRestoredInternalHandler,
	)
	s.cTime = time.Now()
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
	if strings.HasPrefix(uri, s.uriPrefix) {
		uri = strings.TrimPrefix(uri, s.uriPrefix)
	}
	return s.handler.SupportsUri(uri)
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

func (s *ClientService) ProviderInfo() roles.RoleDescriptor {
	return s.ctx.Identity().Describe()
}

func (s *ClientService) HostInfo() roles.RoleDescriptor {
	return s.host.Describe()
}

// returns the corresponding raw uri_trie of the service or ""
func (s *ClientService) matchUri(uri string) (string, error) {
	actualUri := strings.TrimPrefix(uri, s.uriPrefix)
	for _, v := range s.ServiceUris() {
		if strings.HasPrefix(actualUri, v) {
			return v, nil
		}
	}
	return "", errors.New("no matched uri_trie")
}

func (s *ClientService) Handle(message *messages.Message) *messages.Message {
	if strings.HasPrefix(message.Uri(), s.uriPrefix) {
		message = message.Copy().SetUri(strings.TrimPrefix(message.Uri(), s.uriPrefix))
	}
	request := service.NewServiceRequest(message)
	s.serviceTaskQueue.Schedule(request)
	s.m.Track(s.m.GetAssembledTraceId(controllers.TMessagePerformance, message.Id()), "request in queue")
	if s.executionType == service.ServiceExecutionAsync {
		resp := request.Response()
		s.m.Track(s.m.GetAssembledTraceId(controllers.TMessagePerformance, message.Id()), "sync request handled")
		return resp
	} else {
		return nil
	}
}

func (s *ClientService) RegisterRoute(shortUri string, handler service.RequestHandler) (err error) {
	if strings.HasPrefix(shortUri, s.uriPrefix) {
		shortUri = strings.TrimPrefix(shortUri, s.uriPrefix)
	}
	s.withWrite(func() {
		s.serviceUris = append(s.serviceUris, shortUri)
		err = s.handler.Register(shortUri, handler)
	})
	if err != nil {
		return err
	}
	return s.NotifyHostForUpdate()
}

func (s *ClientService) UnregisterRoute(shortUri string) (err error) {
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
		err = s.handler.Unregister(shortUri)
	})
	if err != nil {
		return err
	}
	return s.NotifyHostForUpdate()
}

func (s *ClientService) NotifyHostForUpdate() error {
	if s.Status() == service.ServiceStatusUnregistered {
		return nil
	}
	if s.serviceManagerClient != nil {
		return s.serviceManagerClient.UpdateService(s.Describe())
	}
	return errors.New("no serviceManagerClient found")
}

func (s *ClientService) NewMessage(to string, uri string, msgType int, payload []byte) *messages.Message {
	return messages.DraftMessage(s.ctx.Identity().Id(), to, uri, msgType, payload)
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
	err = s.serviceManagerClient.RegisterService(s.Describe())
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
	err = s.NotifyHostForUpdate()
	if err != nil {
		s.withWrite(func() {
			s.status = service.ServiceStatusRegistered
		})
		return err
	}
	s.withWrite(func() {
		s.status = service.ServiceStatusRunning
	})
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
	s.serviceTaskQueue.Stop()
	// after pool is stopped
	s.withWrite(func() {
		s.status = service.ServiceStatusUnregistered
	})
	return nil
}

func (s *ClientService) unregister() error {
	return s.serviceManagerClient.UnregisterService(s.Describe())
}

func (s *ClientService) HealthCheck() error {
	return s.serviceManagerClient.HealthCheck()
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
	s.KillAllProcessingJobs()
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
	return s.serviceTaskQueue.Cancel(messageId)
}

func (s *ClientService) KillAllProcessingJobs() error {
	return s.serviceTaskQueue.KillAll()
}

func (s *ClientService) CancelAllPendingJobs() error {
	return s.serviceTaskQueue.CancelAll()
}

func (s *ClientService) OnStarted(cb func(service.IBaseService)) {
	s.onStartedCallback = cb
}

func (s *ClientService) OnStopped(cb func(service.IBaseService)) {
	s.onStoppedCallback = cb
}

func (s *ClientService) ResolveByAck(request *service.ServiceRequest) error {
	return request.Resolve(messages.NewACKMessage(request.Id(), s.ProviderInfo().Id, request.From(), request.Uri()))
}

func (s *ClientService) ResolveByResponse(request *service.ServiceRequest, responseData []byte) error {
	return request.Resolve(messages.NewMessage(request.Id(), s.ProviderInfo().Id, request.From(), request.Uri(), messages.MessageTypeServiceResponse, responseData))
}

func (s *ClientService) Init(server roles.ICommonServer, serverConn connection.IConnection) error {
	return errors.New("current service did not implement Init() interface")
}

func (s *ClientService) Logger() *logger.SimpleLogger {
	return s.logger
}
