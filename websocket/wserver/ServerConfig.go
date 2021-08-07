package wserver

import (
	"net/http"
	"wsdk/common/connection"
	common_connection "wsdk/relay_common/connection"
)

type IWsConnectionHandler interface {
	HandleClientConnected(connection.IConnection)
	HandleClientClosed(connection.IConnection, error)
	HandleHTTPRequest(w http.ResponseWriter, r *http.Request)
	HandleConnectionError(connection.IConnection, error)
}

type WsConnectionHandler struct {
	onClientConnected     func(conn connection.IConnection)
	onClientClosed        func(conn connection.IConnection, err error)
	onHttpRequest         func(w http.ResponseWriter, r *http.Request)
	onConnectionError     func(connection.IConnection, error)
	onNoUpgradableRequest func(w http.ResponseWriter, r *http.Request)
}

func (h *WsConnectionHandler) HandleClientConnected(conn connection.IConnection) {
	if h.onClientConnected != nil {
		h.onClientConnected(conn)
	}
}

func (h *WsConnectionHandler) HandleClientClosed(conn connection.IConnection, err error) {
	if h.onClientClosed != nil {
		h.onClientClosed(conn, err)
	}
}

func (h *WsConnectionHandler) HandleHTTPRequest(w http.ResponseWriter, r *http.Request) {
	if h.onHttpRequest != nil {
		h.onHttpRequest(w, r)
	}
}

func (h *WsConnectionHandler) HandleConnectionError(conn connection.IConnection, err error) {
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

func NewWsConnHandler(onClientConnected func(conn connection.IConnection), onClientClosed func(conn connection.IConnection, err error), onHttpRequest func(w http.ResponseWriter, r *http.Request), onConnectionError func(connection.IConnection, error)) *WsConnectionHandler {
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
