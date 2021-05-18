package relay_client

import (
	"wsdk/relay_common"
)

type ClientService struct {
	*relay_common.BaseService
	onDisconnected func(service IClientService)
	onReconnected func(service IClientService)
}

type IClientService interface {
	relay_common.IBaseService
	OnDisconnected(func(service IClientService))
	OnConnectionRestored(func(service IClientService))
}

// TODO impl