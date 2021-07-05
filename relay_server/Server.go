package relay_server

import (
	"sync"
	"wsdk/common/timed"
	"wsdk/relay_common/message_actions"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/roles"
	"wsdk/relay_server/context"
	"wsdk/relay_server/events"
	"wsdk/relay_server/message_dispatcher"
	"wsdk/websocket/connection"
	"wsdk/websocket/wserver"
)

type Server struct {
	*wserver.WServer
	roles.IDescribableRole
	scheduleJobPool         *timed.JobPool
	messageParser           messages.IMessageParser
	messageDispatcher       message_actions.IMessageDispatcher
	clientConnectionHandler IClientConnectionHandler
	lock                    *sync.RWMutex
}

type IServer interface {
	Start() error
	Stop() error
}

func (s *Server) withWrite(cb func()) {
	s.lock.Lock()
	defer s.lock.Unlock()
	cb()
}

func (s *Server) Start() error {
	return s.WServer.Start()
}

func (s *Server) Stop() (closeError error) {
	events.EmitEvent(events.EventServerClosed, "")
	return
}

func (s *Server) handleInitialConnection(conn *connection.WsConnection) {
	s.clientConnectionHandler.HandleConnectionEstablished(conn)
}

func NewServer(identity roles.ICommonServer) *Server {
	server := &Server{
		WServer:           wserver.NewWServer(wserver.NewServerConfig(identity.Id(), identity.Url(), identity.Port(), wserver.DefaultWsConnHandler())),
		IDescribableRole:  identity,
		scheduleJobPool:   context.Ctx.TimedJobPool(),
		messageParser:     messages.NewFBMessageParser(),
		messageDispatcher: message_dispatcher.NewServerMessageDispatcher(),
		lock:              new(sync.RWMutex),
	}
	server.OnClientConnected(server.handleInitialConnection)
	server.clientConnectionHandler = NewClientConnectionHandler(server.messageDispatcher)
	/*
		onHttpRequest func(u func(w http.ResponseWriter, r *http.Handle) error, w http.ResponseWriter, r *http.Handle),
	*/
	return server
}

// TODO
// when server receives a message, after the message is handled, server needs to dispatch the message with messageDispatcher
// server *-- services
// server *-- clients
// service *-- clientConnectionHandler
// when server knows the clientConnectionHandler is disconnected, server should put service in survival mode(constantly health check with clientConnectionHandler id until clientConnectionHandler is recovered or service expired)

/* General health check strategy: clientConnectionHandler should send PING to server every X seconds, and if server does not receive a
   in X + 1 seconds, server will send PING and expect to have a PONG received in X seconds. If that fails, health check
   is considered failed.
*/

/*
func New(id string, description string, ip string, port int) *relay_server {

}
*/
