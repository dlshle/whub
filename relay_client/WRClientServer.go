package relay_client

import "wsdk/relay_common"

type WRClientServer struct {
	*relay_common.WRServer
	services []*relay_common.ServiceDescriptor
}

type IWRClientServer interface {
	// server will decide what services to be returned
	Services() []*relay_common.ServiceDescriptor
	RegisterService()
	UnregisterService()
}
