package relay_client

import (
	"wsdk/relay_common"
	"wsdk/relay_common/messages"
)

type ClientServiceCallback func(message *messages.Message)

type ClientService struct {
	id string
	description string
	serviceUriCallbackMap map[string]ClientServiceCallback
	hostInfo *relay_common.RoleDescriptor
	serviceType int
	accessType int
	executionType int

	onDisconnected func(service IClientService)
	onReconnected func(service IClientService)
}

type IClientService interface {
	Id() string
	Description() string
	ServiceUris() []string
	HostInfo() relay_common.RoleDescriptor
	ServiceType() int
	AccessType() int
	ExecutionType() int

	Register() error
	Start() error
	Stop() error
	Unregister() error

	OnDisconnected(func(service IClientService))
	OnConnectionRestored(func(service IClientService))
}

// TODO impl