package relay_server

import (
	"fmt"
	"net/http"
	"sync"
	"wsdk/common/connection"
	"wsdk/common/logger"
	common_connection "wsdk/relay_common/connection"
	"wsdk/relay_common/message_actions"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/roles"
	"wsdk/relay_server/container"
	"wsdk/relay_server/context"
	"wsdk/relay_server/events"
	server_http "wsdk/relay_server/http"
	"wsdk/relay_server/message_dispatcher"
	"wsdk/relay_server/services"
	"wsdk/relay_server/socket"
	"wsdk/websocket/wserver"
)

type Server struct {
	*wserver.WServer
	roles.ICommonServer
	messageParser           messages.IMessageParser
	messageDispatcher       message_actions.IMessageDispatcher
	clientConnectionHandler socket.IClientConnectionHandler
	httpRequestHandler      server_http.IHTTPRequestHandler
	logger                  *logger.SimpleLogger
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
	// dependency inject to deal w/ IServiceManager
	err := container.Container.Call(services.InitNativeServices)
	if err != nil {
		s.logger.Fatalln("unable to init native services due to ", err)
		return err
	}
	s.logger.Println("all native services have been initialized")
	return s.WServer.Start()
}

func (s *Server) Stop() (closeError error) {
	events.EmitEvent(events.EventServerClosed, "")
	return
}

func (s *Server) handleSocketConnection(conn connection.IConnection, r *http.Request) {
	s.clientConnectionHandler.HandleConnectionEstablished(conn, r)
}

func (s *Server) handleHTTPRequests(w http.ResponseWriter, r *http.Request) {
	s.httpRequestHandler.Handle(w, r)
}

func NewServer(identity roles.ICommonServer) *Server {
	logger := context.Ctx.Logger()
	logger.SetPrefix(fmt.Sprintf("[Server-%s]", identity.Id()))
	context.Ctx.Start(identity)
	wServer := wserver.NewWServer(wserver.NewServerConfig(identity.Id(), identity.Url(), identity.Port(), common_connection.WSConnectionPath, wserver.DefaultWsConnHandler()))
	wServer.SetLogger(logger)
	messageDispatcher := message_dispatcher.NewServerMessageDispatcher()
	// wServer.SetAsyncPool(context.Ctx.AsyncTaskPool())
	server := &Server{
		WServer:            wServer,
		ICommonServer:      identity,
		messageParser:      messages.NewFBMessageParser(),
		messageDispatcher:  messageDispatcher,
		httpRequestHandler: server_http.NewHTTPRequestHandler(messageDispatcher),
		lock:               new(sync.RWMutex),
		logger:             logger,
	}
	server.OnClientConnected(server.handleSocketConnection)
	server.OnNonUpgradableRequest(server.handleHTTPRequests)
	server.SetBeforeUpgradeChecker(server_http.NewWebsocketUpgradeChecker().ShouldUpgradeProtocol)
	server.clientConnectionHandler = socket.NewClientConnectionHandler(server.messageDispatcher)
	/*
		onHttpRequest func(u func(w http.ResponseWriter, r *http.Handle) error, w http.ResponseWriter, r *http.Handle),
	*/
	context.Ctx.Logger().Println("server has been initiated.")
	return server
}
