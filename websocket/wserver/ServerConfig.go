package wserver

import (
	"net/http"
	common_connection "wsdk/relay_common/connection"
	"wsdk/websocket/connection"
)

type IWsConnectionHandler interface {
	HandleClientConnected(*connection.WsConnection)
	HandleClientClosed(*connection.WsConnection, error)
	HandleHTTPRequest(w http.ResponseWriter, r *http.Request)
	HandleConnectionError(*connection.WsConnection, error)
}

type WsConnectionHandler struct {
	onClientConnected     func(conn *connection.WsConnection)
	onClientClosed        func(conn *connection.WsConnection, err error)
	onHttpRequest         func(w http.ResponseWriter, r *http.Request)
	onConnectionError     func(*connection.WsConnection, error)
	onNoUpgradableRequest func(w http.ResponseWriter, r *http.Request)
}

func (h *WsConnectionHandler) HandleClientConnected(conn *connection.WsConnection) {
	if h.onClientConnected != nil {
		h.onClientConnected(conn)
	}
}

func (h *WsConnectionHandler) HandleClientClosed(conn *connection.WsConnection, err error) {
	if h.onClientClosed != nil {
		h.onClientClosed(conn, err)
	}
}

func (h *WsConnectionHandler) HandleHTTPRequest(w http.ResponseWriter, r *http.Request) {
	if h.onHttpRequest != nil {
		h.onHttpRequest(w, r)
	}
}

func (h *WsConnectionHandler) HandleConnectionError(conn *connection.WsConnection, err error) {
	if h.onConnectionError != nil {
		h.onConnectionError(conn, err)
	} else {
		conn.Close()
	}
}

func (h *WsConnectionHandler) HandleNoUpgradableRequest(w http.ResponseWriter, r *http.Request) {
	if h.onNoUpgradableRequest != nil {
		h.onNoUpgradableRequest(w, r)
	} else {
		DefaultNoUpgradableHTTPRequestHandler(w, r)
	}
}

func NewWsConnHandler(onClientConnected func(conn *connection.WsConnection), onClientClosed func(conn *connection.WsConnection, err error), onHttpRequest func(w http.ResponseWriter, r *http.Request), onConnectionError func(*connection.WsConnection, error)) *WsConnectionHandler {
	return &WsConnectionHandler{onClientConnected: onClientConnected, onClientClosed: onClientClosed, onHttpRequest: onHttpRequest, onConnectionError: onConnectionError}
}

func DefaultWsConnHandler() *WsConnectionHandler {
	return NewWsConnHandler(nil, nil, nil, nil)
}

type WsServerConfig struct {
	Name    string
	Address string
	Port    int
	*WsConnectionHandler
}

func NewServerConfig(name string, address string, port int, handler *WsConnectionHandler) WsServerConfig {
	return WsServerConfig{name, address, port, handler}
}

func DefaultHTTPRequestHandler(u func(w http.ResponseWriter, r *http.Request) error, w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != common_connection.WSConnectionPath {
		code := http.StatusInternalServerError
		statusText := http.StatusText(code)
		// log path err
		http.Error(w, statusText, code)
		return
	}
	err := u(w, r)
	if err != nil {
		// log err
		return
	}
}

func DefaultNoUpgradableHTTPRequestHandler(w http.ResponseWriter, r *http.Request) {
	code := http.StatusInternalServerError
	statusText := http.StatusText(code)
	// log path err
	http.Error(w, statusText, code)
}
