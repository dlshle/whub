package wserver

import (
	"net/http"
	"wsdk/common/connection"
)

type IWsConnectionHandler interface {
	HandleClientConnected(connection.IConnection, map[string][]string)
	HandleClientClosed(connection.IConnection, error)
	HandleConnectionError(connection.IConnection, error)
}

type WsConnectionHandler struct {
	onClientConnected     func(conn connection.IConnection, header map[string][]string)
	onClientClosed        func(conn connection.IConnection, err error)
	onConnectionError     func(connection.IConnection, error)
	onNoUpgradableRequest func(w http.ResponseWriter, r *http.Request)
	beforeUpgradeChecker  func(r *http.Request) error
}

func (h *WsConnectionHandler) HandleClientConnected(conn connection.IConnection, header map[string][]string) {
	if h.onClientConnected != nil {
		h.onClientConnected(conn, header)
	}
}

func (h *WsConnectionHandler) HandleClientClosed(conn connection.IConnection, err error) {
	if h.onClientClosed != nil {
		h.onClientClosed(conn, err)
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

func (h *WsConnectionHandler) CheckUpgradeRequest(r *http.Request) error {
	if h.beforeUpgradeChecker != nil {
		return h.beforeUpgradeChecker(r)
	}
	return nil
}

func NewWsConnHandler(onClientConnected func(conn connection.IConnection, header map[string][]string), onClientClosed func(conn connection.IConnection, err error), onConnectionError func(connection.IConnection, error)) *WsConnectionHandler {
	return &WsConnectionHandler{onClientConnected: onClientConnected, onClientClosed: onClientClosed, onConnectionError: onConnectionError}
}

func DefaultWsConnHandler() *WsConnectionHandler {
	return NewWsConnHandler(nil, nil, nil)
}

type WsServerConfig struct {
	Name           string
	Address        string
	Port           int
	UpgradeUrlPath string
	*WsConnectionHandler
}

func NewServerConfig(name string, address string, port int, upgradeUrlPath string, handler *WsConnectionHandler) WsServerConfig {
	return WsServerConfig{name, address, port, upgradeUrlPath, handler}
}

func DefaultNoUpgradableHTTPRequestHandler(w http.ResponseWriter, r *http.Request) {
	code := http.StatusInternalServerError
	statusText := http.StatusText(code)
	// log path err
	http.Error(w, statusText, code)
}
