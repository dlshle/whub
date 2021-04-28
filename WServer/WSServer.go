package WServer

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"wsdk/Common"

	"github.com/dlshle/gommon/logger"
	"github.com/gorilla/websocket"
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
	wsServer.logger = logger.New(os.Stdout, "[WServer]", true)
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

func (w *WServer) Start() (err error) {
	w.listener, err = net.Listen("tcp", w.address)
	if err != nil {
		w.logger.Println("net listen error:", err)
		return
	}
	err = http.Serve(w.listener, w)
	if err != nil {
		w.logger.Println("http serve error:", err)
		return
	}
	return nil
}

func (ws *WServer) handleNewConnection(conn *websocket.Conn) {
	c := Common.NewWsConnection(conn, nil, nil, nil)
	defer c.Close()
	c.OnClose(func(err error) { ws.handler.OnClientClosed(c, err) })
	c.OnError(func(err error) { ws.handler.OnConnectionError(c, err) })
	ws.handler.OnClientConnected(c)
}

func (ws *WServer) Send(conn *Common.WsConnection, data []byte) error {
	// log send
	return conn.Write(data)
}

func (ws *WServer) OnConnectionError(cb func(*Common.WsConnection, error)) {
	ws.handler.onConnectionError = cb
}

func (ws *WServer) OnClientConnected(cb func(*Common.WsConnection)) {
	ws.handler.onClientConnected = cb
}

func (ws *WServer) OnClientClosed(cb func(*Common.WsConnection, error)) {
	ws.handler.onClientClosed = cb
}

func (ws *WServer) OnHttpRequest(cb func(upgradeFunc func(w http.ResponseWriter, r *http.Request) error, w http.ResponseWriter, r *http.Request)) {
	ws.handler.onHttpRequest = cb
}