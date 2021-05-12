package relay_server

import (
	"encoding/json"
	"fmt"
	"github.com/dlshle/gommon/timed"
	"sync"
	"time"
	"wsdk/base/common"
	"wsdk/base/wserver"
	"wsdk/relay_common"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/utils"
)

const (
	MaxServicePerClient = 5
	ServiceKillTimeout = time.Minute * 15
)

type WRelayServer struct {
	ctx *relay_common.WRContext
	*wserver.WServer
	relay_common.IDescribableRole
	anonymousClient map[string]*WRServerClient // raw clients or pure anony clients
	clients map[string]*WRServerClient
	serviceMap map[string]IServerService // serviceId <--> ServerService when a client is closed, should also kill the service until it's expired(Tdead + Texipre_period)
	serviceExpirePeriod time.Duration
	scheduleJobPool *timed.JobPool
	messageHandler messages.IMessageHandler
	messageDispatcher messages.IMessageDispatcher
	lock *sync.RWMutex
}

type IWRelayServer interface {
	Start() error
	Stop() error
	RegisterService(IServerService) error
	UnregisterService(string) error

	HasClient(id string) bool
	GetClient(clientId string) *WRServerClient
	HasService(id string) bool
	GetService(id string) IServerService

	GetServicesByClientId(id string) ([]IServerService, error)
}

type clientExtraInfoDescriptor struct {
	pScope int
	cKey string
	cType int
}

func (s *WRelayServer) withWrite(cb func()) {
	s.lock.Lock()
	defer s.lock.Unlock()
	cb()
}

func (s *WRelayServer) Start() error {
	return s.WServer.Start()
}

func (s *WRelayServer) Stop() (closeError error) {
	errorMsg := ""
	hasErr := false
	// safe close server
	for _, c := range s.anonymousClient {
		if err := c.Close(); err != nil {
			hasErr = true
			errorMsg += err.Error() + "\n"
		}
	}
	for _, c := range s.clients {
		if err := c.Close(); err != nil {
			hasErr = true
			errorMsg += err.Error() + "\n"
		}
	}
	if hasErr {
		closeError = NewServerCloseFailError(errorMsg)
	}
	return
}

func (s *WRelayServer) GetClient(id string) *WRServerClient {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.clients[id]
}

func (s *WRelayServer) HasClient(id string) bool {
	return s.GetClient(id) != nil
}

func (s *WRelayServer) GetService(id string) IServerService {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.serviceMap[id]
}

func (s *WRelayServer) HasService(id string) bool {
	return s.GetService(id) != nil
}

func (s *WRelayServer) cancelTimedJob(jobId int64) bool {
	return s.scheduleJobPool.CancelJob(jobId)
}

func (s *WRelayServer) scheduleTimeoutJob(job func()) int64 {
	return s.scheduleJobPool.ScheduleAsyncTimeoutJob(job, ServiceKillTimeout)
}

func (s *WRelayServer) getServicesByClientId(id string) []IServerService {
	services := make([]IServerService, 0, MaxServicePerClient)
	s.lock.RLock()
	defer s.lock.RUnlock()
	for _, v := range s.serviceMap {
		if v.Provider().Id() == id {
			services = append(services, v)
		}
	}
	return services
}

func (s *WRelayServer) withServicesFromClientId(clientId string, cb func([]IServerService)) error {
	services, err := s.GetServicesByClientId(clientId)
	if err != nil {
		return err
	}
	cb(services)
	return nil
}

func (s *WRelayServer) unregisterAllServicesFromClientId(clientId string) error {
	return s.withServicesFromClientId(clientId, func(services []IServerService) {
		for i, _ := range services {
			if services[i] != nil {
				s.unregisterService(services[i].Id())
			}
		}
	})
}

// nil -> no such client, [] -> no service
func (s *WRelayServer) GetServicesByClientId(id string) ([]IServerService, error) {
	if !s.HasClient(id) {
		return nil, NewNoSuchClientError(id)
	}
	return s.getServicesByClientId(id), nil
}

func (s *WRelayServer) serviceCountByClientId(id string) int {
	return len(s.getServicesByClientId(id))
}

func (s *WRelayServer) RegisterService(service IServerService) error {
	return s.registerService(service)
}

func (s *WRelayServer) registerService(service IServerService) error {
	clientId := service.Provider().Id()
	if !s.HasClient(clientId) {
		return NewNoSuchClientError(clientId)
	}
	if s.serviceCountByClientId(clientId) >= MaxServicePerClient {
		return NewClientExceededMaxServiceCountError(clientId)
	}
	var serviceDeadTimeoutJobId int64 = -1
	s.withWrite(func() {
		service.OnHealthCheckFails(func(service IServerService) {
			// log
			service.KillAllProcessingJobs()
			// schedule timeout job to really kill the service if it's been dead for X duration
			serviceDeadTimeoutJobId = s.scheduleTimeoutJob(func() {
				s.unregisterService(service.Id())
			})
		})
		service.OnHealthRestored(func(service IServerService) {
			// log
			if serviceDeadTimeoutJobId > -1 {
				s.cancelTimedJob(serviceDeadTimeoutJobId)
			}
		})
		s.serviceMap[service.Id()] = service
	})
	return nil
}

func (s *WRelayServer) UnregisterService(serviceId string) error {
	return s.unregisterService(serviceId)
}

func (s *WRelayServer) unregisterService(serviceId string) error {
	if !s.HasService(serviceId) {
		return NewNoSuchServiceError(serviceId)
	}
	s.withWrite(func() {
		s.serviceMap[serviceId].Stop()
		delete(s.serviceMap, serviceId)
	})
	return nil
}

func (s *WRelayServer) handleInitialConnection(conn *common.WsConnection) {
	rawConn := connection.NewWRConnection(conn, connection.DefaultTimeout, s.messageHandler, s.ctx.NotificationEmitter())
	// any message from any connection needs to go through here
	rawConn.OnAnyMessage(func(message *messages.Message) {
		if s.messageDispatcher != nil {
			s.messageDispatcher.Dispatch(message)
		}
	})
	rawClient := s.NewAnonymousClient(rawConn)
	s.withWrite(func() {
		s.anonymousClient[conn.Address()] = rawClient
	})
	resp, err := rawClient.Request(rawClient.NewMessage(s.Id(), messages.MessageTypeServerDescriptor, ([]byte)(s.Describe().String())))
	// try to handle anonymous client upgrade
	if err == nil && resp.MessageType() == messages.MessageTypeClientDescriptor {
		var clientDescriptor relay_common.RoleDescriptor
		var clientExtraInfo clientExtraInfoDescriptor
		err = utils.ProcessWithError([]func()error{
			func() error {
				return json.Unmarshal(resp.Payload(), &clientDescriptor)
			},
			func() error {
				return json.Unmarshal(([]byte)(clientDescriptor.ExtraInfo), &clientExtraInfo)
			},
		})
		if err == nil {
			s.withWrite(func() {
				delete(s.anonymousClient, conn.Address())
				client := s.NewClient(rawClient.WRConnection, clientDescriptor.Id, clientDescriptor.Description, clientExtraInfo.cType, clientExtraInfo.cKey, clientExtraInfo.pScope)
				s.clients[clientDescriptor.Id] = client
				s.initClientCallbackHandlers(client)
			})
			err = s.tryToRestoreDeadServicesFromReconnectedClient(clientDescriptor.Id)
			// log err
		}
	}
}

func (s *WRelayServer) tryToRestoreDeadServicesFromReconnectedClient(clientId string) (err error) {
	s.withServicesFromClientId(clientId, func(services []IServerService) {
		client := s.GetClient(clientId)
		if client == nil {
			err = NewNoSuchClientError(clientId)
			return
		}
		for i, _ := range services {
			if services[i] != nil {
				if err = services[i].RestoreExternally(client); err != nil {
					return
				}
			}
		}
	})
	return
}

// client connection close handler is defined in the upgrade part ^^
func (s *WRelayServer) handleAnonymousConnectionClosed(c *common.WsConnection, err error) {
	conn := s.anonymousClient[c.Address()]
	fmt.Println(conn, " closed")
}

func (s *WRelayServer) initClientCallbackHandlers(client *WRServerClient) {
	client.OnClose(func(err error) {
		s.handleClientConnectionClosed(client, err)
	})
	client.OnError(func(err error) {
		s.handleClientError(client, err)
	})
}

func (s *WRelayServer) handleClientConnectionClosed(c *WRServerClient, err error) {
	if err == nil {
		// normal closure
		// close all services
		s.unregisterAllServicesFromClientId(c.Id())
		// remove client from connection
		s.withWrite(func() {
			delete(s.clients, c.Id())
		})
	} else {
		// unexpected closure
		// service should kill all jobs and transit to DeadMode automatically
		s.withWrite(func() {
			delete(s.clients, c.Id())
		})
	}
}

func (s *WRelayServer) handleClientError(c *WRServerClient, err error) {
	// log
	fmt.Printf("Server(%s) error(%s)", c.Id(), err.Error())
}

func (s *WRelayServer) NewClient(conn *connection.WRConnection, id string, description string, cType int, cKey string, pScope int) *WRServerClient {
	return NewClient(s.ctx, conn, id, description, cType, cKey, pScope)
}

func (s *WRelayServer) NewAnonymousClient(conn *connection.WRConnection) *WRServerClient {
	return NewAnonymousClient(s.ctx, conn)
}

func NewServer(ctx *relay_common.WRContext, port int) *WRelayServer {
	server := &WRelayServer{
		ctx: ctx,
		WServer: wserver.NewWServer(wserver.NewServerConfig(ctx.Identity().Id(), "127.0.0.1", port, wserver.DefaultWsConnHandler())),
		IDescribableRole: ctx.Identity(),
		anonymousClient: make(map[string]*WRServerClient),
		clients: make(map[string]*WRServerClient),
		serviceMap: make(map[string]IServerService),
		serviceExpirePeriod: time.Second,
		scheduleJobPool: ctx.TimedJobPool(),
		messageHandler: messages.NewSimpleMessageHandler(),
		lock: new(sync.RWMutex),
	}
	server.OnClientConnected(server.handleInitialConnection)
	/*
		onHttpRequest func(u func(w http.ResponseWriter, r *http.Request) error, w http.ResponseWriter, r *http.Request),
	 */
	return server
}

// TODO
// when server receives a message, after the message is handled, server needs to dispatch the message with messageDispatcher
// server *-- services
// server *-- clients
// service *-- client
// when server knows the client is disconnected, server should put service in survival mode(constantly health check with client id until client is recovered or service expired)

/* General health check strategy: client should send PING to server every X seconds, and if server does not receive a
   in X + 1 seconds, server will send PING and expect to have a PONG received in X seconds. If that fails, health check
   is considered failed.
*/

/*
func New(id string, description string, ip string, port int) *relay_server {

}
*/
