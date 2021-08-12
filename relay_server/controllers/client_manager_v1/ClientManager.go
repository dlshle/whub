package client_manager_v1

import (
	"wsdk/relay_common/connection"
	"wsdk/relay_server/client"
)

type IClientManager interface {
	HasClient(id string) bool
	GetClient(id string) *client.Client
	GetClientByAddr(addr string) *client.Client
	WithAllClients(cb func(clients []*client.Client))
	DisconnectClient(id string) error
	DisconnectClientByAddr(addr string) error
	DisconnectAllClients() error
	AcceptClient(id string, client *client.Client, conn connection.IConnection) error
}

type ClientManager struct {
}
