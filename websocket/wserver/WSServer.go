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
	name           string
	address        string
	listener       net.Listener
	upgrader       *websocket.Upgrader
	handler        *WsConnectionHandler
	logger         *logger.SimpleLogger
	upgradeUrlPath string
}

func NewWServer(config WsServerConfig) *WServer {
	name := config.Name
	address := config.Address
	port := config.Port
	wsServer := &WServer{}
	wsServer.logger = logger.New(os.Stdout, "[wserver]", true)
	wsServer.name = name
	wsServer.address = fmt.Sprintf("%s:%d", address, port)
	if config.UpgradeUrlPath == "" {
		config.UpgradeUrlPath = common_connection.WSConnectionPath
	}
	wsServer.upgradeUrlPath = config.UpgradeUrlPath
	wsServer.upgrader = &websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 1024,
		CheckOrigin: func(req *http.Request) bool {
			if req.Method != "GET" {
				wsServer.logger.Printf("invalid request from %s(METHOD = %s URL = %s)\n", req.RemoteAddr, req.Method, req.URL)
				return false
			}
			if req.URL.Path != wsServer.upgradeUrlPath {
				wsServer.logger.Printf("invalid path from %s(METHOD = %s URL = %s)\n", req.RemoteAddr, req.Method, req.URL)
				return false
			}
			return true
		},
	}
	wsServer.handler = config.WsConnectionHandler
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
	ws.handleNewConnection(conn, r.Header)
	return
}

func (ws *WServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ws.handleHTTPRequest(w, r)
}

func (ws *WServer) Start() (err error) {
	// basically same flow as how TCP server starts, so can we abstract the start logic, and use the same listener?
	// no bitch, you can't unless you can write your own http handler for a listener! Clearly, you can't so fuck it.
	// detailed reason: http.Server takes over the listener, and it will keep accepting from it, so when other goroutine
	// can not use the listener or there will be race condition as TCPListener does not use lock internally?
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

func (ws *WServer) handleUpgradeFailure(w http.ResponseWriter, message string) {
	w.WriteHeader(http.StatusBadRequest)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Write([]byte(fmt.Sprintf("{\"message\":\"unable to upgrade HTTP request due to %s\"", message)))
}

func (ws *WServer) handleHTTPRequest(w http.ResponseWriter, r *http.Request) {
	// each HTTP request is a new goroutine, so no need to add extra concurrency here
	if r.URL.Path != ws.upgradeUrlPath {
		ws.handler.HandleNoUpgradableRequest(w, r)
		return
	}
	err := ws.handler.CheckUpgradeRequest(r)
	if err != nil {
		ws.handleUpgradeFailure(w, err.Error())
		return
	}
	err = ws.upgradeHTTP(w, r)
	if err != nil {
		ws.logger.Printf("err while upgrading HTTP request: %s", err.Error())
		return
	}
}

func (ws *WServer) handleNewConnection(conn *websocket.Conn, header map[string][]string) {
	ws.logger.Printf("new connection from %s detected", conn.RemoteAddr())
	c := connection.NewWsConnection(conn, nil, nil, nil)
	defer c.Close()
	c.OnClose(func(err error) { ws.handler.HandleClientClosed(c, err) })
	c.OnError(func(err error) { ws.handler.HandleConnectionError(c, err) })
	ws.handler.HandleClientConnected(c, header)
}

func (ws *WServer) OnConnectionError(cb func(base_conn.IConnection, error)) {
	ws.handler.onConnectionError = cb
}

func (ws *WServer) OnClientConnected(cb func(base_conn.IConnection, map[string][]string)) {
	ws.handler.onClientConnected = cb
}

func (ws *WServer) OnClientClosed(cb func(base_conn.IConnection, error)) {
	ws.handler.onClientClosed = cb
}

func (ws *WServer) OnNonUpgradableRequest(cb func(w http.ResponseWriter, r *http.Request)) {
	ws.handler.onNoUpgradableRequest = cb
}

func (ws *WServer) SetBeforeUpgradeChecker(checker func(r *http.Request) error) {
	ws.handler.beforeUpgradeChecker = checker
}

func (ws *WServer) SetLogger(logger *logger.SimpleLogger) {
	ws.logger = logger
}
