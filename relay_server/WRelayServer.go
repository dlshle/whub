package relay_server

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
	"wsdk/base/connection"
	"wsdk/base/wserver"
	"wsdk/common/timed"
	"wsdk/relay_common"
	common_connection "wsdk/relay_common/connection"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/utils"
	"wsdk/relay_server/service"
)

const (
	MaxServicePerClient = 5
	ServiceKillTimeout  = time.Minute * 15
)

type WRelayServer struct {
	ctx *Context
	*wserver.WServer
	relay_common.IDescribableRole
	anonymousClient map[string]*WRServerClient // raw clients or pure anony clients
	IClientManager
	service.IServiceManager
	scheduleJobPool   *timed.JobPool
	messageParser     messages.IMessageParser
	messageDispatcher messages.IMessageDispatcher
	lock              *sync.RWMutex
}

type IWRelayServer interface {
	service.IServiceManager
	IClientManager
	Start() error
	Stop() error
}

type clientExtraInfoDescriptor struct {
	pScope int
	cKey   string
	cType  int
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
	errorMsg += s.UnregisterAllServices().Error()
	errorMsg += s.DisconnectAllClients().Error()
	for _, c := range s.anonymousClient {
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

func (s *WRelayServer) handleInitialConnection(conn *connection.WsConnection) {
	rawConn := common_connection.NewWRConnection(conn, common_connection.DefaultTimeout, s.messageParser, s.ctx.NotificationEmitter())
	// any message from any connection needs to go through here
	rawConn.OnIncomingMessage(func(message *messages.Message) {
		if s.messageDispatcher != nil {
			s.messageDispatcher.Dispatch(message, rawConn)
		}
	})
	rawClient := s.createAnonymousClient(rawConn)
	s.withWrite(func() {
		s.anonymousClient[conn.Address()] = rawClient
	})
	resp, err := rawClient.Request(rawClient.NewMessage(s.Id(), "", messages.MessageTypeServerDescriptor, ([]byte)(s.Describe().String())))
	// try to handle anonymous client upgrade
	// TODO maybe do this in client manager??? Need to somehow relate this to RelayManagementService
	if err == nil && resp.MessageType() == messages.MessageTypeClientDescriptor {
		var clientDescriptor relay_common.RoleDescriptor
		var clientExtraInfo clientExtraInfoDescriptor
		err = utils.ProcessWithError([]func() error{
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
			})
			client := s.createClient(rawClient.WRConnection, clientDescriptor.Id, clientDescriptor.Description, clientExtraInfo.cType, clientExtraInfo.cKey, clientExtraInfo.pScope)
			s.AddClient(client)
			s.initClientCallbackHandlers(client)
			// log err
		}
	}
}

// client connection close handler is defined in the upgrade part ^^
func (s *WRelayServer) handleAnonymousConnectionClosed(c *connection.WsConnection, err error) {
	conn := s.anonymousClient[c.Address()]
	fmt.Println(conn, " closed")
}

func (s *WRelayServer) initClientCallbackHandlers(client *WRServerClient) {
	client.OnClose(func(err error) {
		s.HandleClientConnectionClosed(client, err)
	})
	client.OnError(func(err error) {
		s.HandleClientError(client, err)
	})
}

func (s *WRelayServer) createClient(conn *common_connection.WRConnection, id string, description string, cType int, cKey string, pScope int) *WRServerClient {
	return NewClient(s.ctx, conn, id, description, cType, cKey, pScope)
}

func (s *WRelayServer) createAnonymousClient(conn *common_connection.WRConnection) *WRServerClient {
	return NewAnonymousClient(s.ctx, conn)
}

func NewServer(ctx *Context, port int) *WRelayServer {
	server := &WRelayServer{
		ctx:              ctx,
		WServer:          wserver.NewWServer(wserver.NewServerConfig(ctx.Identity().Id(), "127.0.0.1", port, wserver.DefaultWsConnHandler())),
		IDescribableRole: ctx.Identity(),
		anonymousClient:  make(map[string]*WRServerClient),
		IClientManager:   NewClientManager(ctx),
		IServiceManager:  service.NewServiceManager(ctx),
		scheduleJobPool:  ctx.TimedJobPool(),
		messageParser:    messages.NewSimpleMessageParser(),
		lock:             new(sync.RWMutex),
	}
	server.OnClientConnected(server.handleInitialConnection)
	/*
		onHttpRequest func(u func(w http.ResponseWriter, r *http.Handle) error, w http.ResponseWriter, r *http.Handle),
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
