package hub_server

import (
	"fmt"
	"net/http"
	"whub/common/connection"
	"whub/common/logger"
	common_connection "whub/hub_common/connection"
	"whub/hub_common/dispatcher"
	"whub/hub_common/roles"
	"whub/hub_server/context"
	"whub/hub_server/events"
	server_http "whub/hub_server/http"
	"whub/hub_server/message_dispatcher"
	"whub/hub_server/modules"
	"whub/hub_server/services"
	"whub/hub_server/socket"
	"whub/websocket/wserver"
)

type Server struct {
	*wserver.WServer
	roles.ICommonServer
	messageDispatcher       dispatcher.IMessageDispatcher
	clientConnectionHandler socket.ISocketConnectionHandler
	httpRequestHandler      server_http.IHTTPRequestHandler
	logger                  *logger.SimpleLogger
}

type IServer interface {
	Start() error
	Stop() error
}

func (s *Server) Start() (err error) {
	err = services.InitNativeServices()
	if err != nil {
		s.logger.Fatalln("unable to init native services due to ", err.Error())
		return err
	}
	s.logger.Println("all native services have been initialized")

	s.logger.Println("message dispatcher and http request handler has been initialized")
	return s.WServer.Start()
}

func (s *Server) Stop() (closeError error) {
	events.EmitEvent(events.EventServerClosed, "")
	s.WServer.Stop()
	return
}

func (s *Server) handleSocketConnection(conn connection.IConnection, r *http.Request) {
	s.clientConnectionHandler.HandleConnectionEstablished(conn, r)
}

func (s *Server) handleHTTPRequests(w http.ResponseWriter, r *http.Request) {
	s.httpRequestHandler.Handle(w, r)
}

func NewServer(identity roles.ICommonServer, websocketPath string) *Server {
	if websocketPath == "" {
		websocketPath = common_connection.WSConnectionPath
	}
	logger := context.Ctx.Logger()
	logger.SetPrefix(fmt.Sprintf("[Server-%s]", identity.Id()))
	context.Ctx.Start(identity)
	wServer := wserver.NewWServer(wserver.NewServerConfig(identity.Id(), identity.Url(), identity.Port(), websocketPath, wserver.DefaultWsConnHandler()))
	wServer.SetLogger(logger)
	err := modules.InitCoreComponents()
	if err != nil {
		logger.Fatalln("unable to load modules components due to ", err.Error())
		panic(err)
	}
	messageDispatcher := message_dispatcher.NewServerMessageDispatcher()
	server := &Server{
		WServer:            wServer,
		ICommonServer:      identity,
		messageDispatcher:  messageDispatcher,
		httpRequestHandler: server_http.NewHTTPRequestHandler(messageDispatcher),
		logger:             logger,
	}
	server.OnClientConnected(server.handleSocketConnection)
	server.OnNonUpgradableRequest(server.handleHTTPRequests)
	server.SetBeforeUpgradeChecker(server_http.NewWebsocketUpgradeChecker().ShouldUpgradeProtocol)
	server.clientConnectionHandler = socket.NewSocketConnectionHandler(server.messageDispatcher)
	/*
		onHttpRequest func(u func(w http.ResponseWriter, r *http.Handle) error, w http.ResponseWriter, r *http.Handle),
	*/
	context.Ctx.Logger().Printf("server has been initiated on %s:%d with websocket path %s", identity.Url(), identity.Port(), websocketPath)
	return server
}
