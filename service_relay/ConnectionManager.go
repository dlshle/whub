package service_relay

import "wsdk/relay_common/connection"

type clientConnections struct {
	clientId string
	conns    []connection.IConnection
}

type ConnectionManager struct {
	serviceConnections map[string]*clientConnections
}
