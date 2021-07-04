package wserver

import (
	"fmt"
	"github.com/gorilla/websocket"
	"net"
	"net/http"
	"os"
	"wsdk/common/logger"
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
			if req.URL.Path != "/ws" {
				wsServer.logger.Printf("invalid path from %s(METHOD = %s URL = %s)\n", req.RemoteAddr, req.Method, req.URL)
				return false
			}
			return true
		},
	}
	wsServer.handler = config.WsConnectionHandler
	return wsServer
}

func (ws *WServer) handleUpgrade(w http.ResponseWriter, r *http.Request) (err error) {
	conn, err := ws.upgrader.Upgrade(w, r, nil)
	go ws.handleNewConnection(conn)
	return
}

func (ws *WServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ws.handler.OnHttpRequest(ws.handleUpgrade, w, r)
}

func (ws *WServer) Start() (err error) {
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

func (ws *WServer) handleNewConnection(conn *websocket.Conn) {
	c := connection.NewWsConnection(conn, nil, nil, nil)
	defer c.Close()
	c.OnClose(func(err error) { ws.handler.OnClientClosed(c, err) })
	c.OnError(func(err error) { ws.handler.OnConnectionError(c, err) })
	ws.handler.OnClientConnected(c)
}

func (ws *WServer) Send(conn *connection.WsConnection, data []byte) error {
	// log send
	return conn.Write(data)
}

func (ws *WServer) OnConnectionError(cb func(*connection.WsConnection, error)) {
	ws.handler.onConnectionError = cb
}

func (ws *WServer) OnClientConnected(cb func(*connection.WsConnection)) {
	ws.handler.onClientConnected = cb
}

func (ws *WServer) OnClientClosed(cb func(*connection.WsConnection, error)) {
	ws.handler.onClientClosed = cb
}

func (ws *WServer) OnHttpRequest(cb func(upgradeFunc func(w http.ResponseWriter, r *http.Request) error, w http.ResponseWriter, r *http.Request)) {
	ws.handler.onHttpRequest = cb
}
