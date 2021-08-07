package wserver

import (
	"fmt"
	"github.com/gorilla/websocket"
	"net"
	"net/http"
	"os"
	base_conn "wsdk/common/connection"
	"wsdk/common/logger"
	common_connection "wsdk/relay_common/connection"
	"wsdk/websocket/connection"
)

type WServer struct {
	name     string
	address  string
	listener net.Listener
	upgrader *websocket.Upgrader
	handler  *WsConnectionHandler
	logger   *logger.SimpleLogger
}

func NewWServer(config WsServerConfig) *WServer {
	name := config.Name
	address := config.Address
	port := config.Port
	wsServer := &WServer{}
	wsServer.logger = logger.New(os.Stdout, "[wserver]", true)
	wsServer.name = name
	wsServer.address = fmt.Sprintf("%s:%d", address, port)
	wsServer.upgrader = &websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 1024,
		CheckOrigin: func(req *http.Request) bool {
			if req.Method != "GET" {
				wsServer.logger.Printf("invalid request from %s(METHOD = %s URL = %s)\n", req.RemoteAddr, req.Method, req.URL)
				return false
			}
			if req.URL.Path != common_connection.WSConnectionPath {
				wsServer.logger.Printf("invalid path from %s(METHOD = %s URL = %s)\n", req.RemoteAddr, req.Method, req.URL)
				return false
			}
			return true
		},
	}
	wsServer.handler = config.WsConnectionHandler
	wsServer.OnHttpRequest(wsServer.handleHTTPRequest)
	return wsServer
}

func (ws *WServer) upgradeHTTP(w http.ResponseWriter, r *http.Request) (err error) {
	conn, err := ws.upgrader.Upgrade(w, r, nil)
	// probably not necessary to do this on a different goroutine
	/*
		if ws.asyncPool != nil {
			ws.asyncPool.Schedule(func() { ws.handleNewConnection(conn) })
		} else {
			go ws.handleNewConnection(conn)
		}
	*/
	ws.handleNewConnection(conn)
	return
}

func (ws *WServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ws.handler.HandleHTTPRequest(w, r)
}

func (ws *WServer) Start() (err error) {
	ws.logger.Println("starting ws server...")
	ws.listener, err = net.Listen("tcp", ws.address)
	if err != nil {
		ws.logger.Println("net listen error:", err)
		return
	}
	err = http.Serve(ws.listener, ws)
	if err != nil {
		ws.logger.Println("http serve error:", err)
		return
	}
	return nil
}

func (ws *WServer) Stop() (err error) {
	return ws.listener.Close()
}

func (ws *WServer) handleHTTPRequest(w http.ResponseWriter, r *http.Request) {
	// each HTTP request is a new goroutine, so no need to add extra concurrency here
	if r.URL.Path != common_connection.WSConnectionPath {
		ws.handler.HandleNoUpgradableRequest(w, r)
		return
	}
	err := ws.upgradeHTTP(w, r)
	if err != nil {
		ws.logger.Printf("err while upgrading HTTP request: %s", err.Error())
		return
	}
}

func (ws *WServer) handleNewConnection(conn *websocket.Conn) {
	ws.logger.Printf("new connection from %s detected", conn.RemoteAddr())
	c := connection.NewWsConnection(conn, nil, nil, nil)
	defer c.Close()
	c.OnClose(func(err error) { ws.handler.HandleClientClosed(c, err) })
	c.OnError(func(err error) { ws.handler.HandleConnectionError(c, err) })
	ws.handler.HandleClientConnected(c)
}

func (ws *WServer) OnConnectionError(cb func(base_conn.IConnection, error)) {
	ws.handler.onConnectionError = cb
}

func (ws *WServer) OnClientConnected(cb func(base_conn.IConnection)) {
	ws.handler.onClientConnected = cb
}

func (ws *WServer) OnClientClosed(cb func(base_conn.IConnection, error)) {
	ws.handler.onClientClosed = cb
}

func (ws *WServer) OnHttpRequest(cb func(w http.ResponseWriter, r *http.Request)) {
	ws.handler.onHttpRequest = cb
}

func (ws *WServer) OnNonUpgradableRequest(cb func(w http.ResponseWriter, r *http.Request)) {
	ws.handler.onNoUpgradableRequest = cb
}

func (ws *WServer) SetLogger(logger *logger.SimpleLogger) {
	ws.logger = logger
}
