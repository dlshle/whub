package wserver

import (
	"net/http"
	"wsdk/base/common"
)

type IWsConnectionHandler interface {
	OnClientConnected(*common.WsConnection)
	OnClientClosed(*common.WsConnection, error)
	OnHttpRequest(upgradeFunc func(w http.ResponseWriter, r *http.Request) error, w http.ResponseWriter, r *http.Request)
	OnConnectionError(*common.WsConnection, error)
}

type WsConnectionHandler struct {
	onClientConnected func(conn *common.WsConnection)
	onClientClosed    func(conn *common.WsConnection, err error)
	onHttpRequest     func(u func(w http.ResponseWriter, r *http.Request) error, w http.ResponseWriter, r *http.Request)
	onConnectionError func(*common.WsConnection, error)
}

func (h *WsConnectionHandler) OnClientConnected(conn *common.WsConnection) {
	if h.onClientConnected != nil {
		h.onClientConnected(conn)
	}
}

func (h *WsConnectionHandler) OnClientClosed(conn *common.WsConnection, err error) {
	if h.onClientClosed != nil {
		h.onClientClosed(conn, err)
	}
}

func (h *WsConnectionHandler) OnHttpRequest(u func(w http.ResponseWriter, r *http.Request) error, w http.ResponseWriter, r *http.Request) {
	if h.onHttpRequest != nil {
		h.onHttpRequest(u, w, r)
	}
}

func (h *WsConnectionHandler) OnConnectionError(conn *common.WsConnection, err error) {
	if h.onConnectionError != nil {
		h.onConnectionError(conn, err)
	}
}

func NewWsConnHandler(onClientConnected func(conn *common.WsConnection), onClientClosed func(conn *common.WsConnection, err error), onHttpRequest func(u func(w http.ResponseWriter, r *http.Request) error, w http.ResponseWriter, r *http.Request), onConnectionError func(*common.WsConnection, error)) *WsConnectionHandler {
	if onHttpRequest == nil {
		onHttpRequest = DefaultHTTPRequestHandler
	}
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
	if r.URL.Path != "/ws" {
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
