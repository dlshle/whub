package relay_client

import (
	"wsdk/relay_common"
	"wsdk/relay_common/service"
)

type WRClientServer struct {
	*relay_common.WRServer
	services []*service.ServiceDescriptor
}

type IWRClientServer interface {
	// server will decide what services to be returned
	Services() []*service.ServiceDescriptor
	RegisterService()
	UnregisterService()
}

// TODO impl