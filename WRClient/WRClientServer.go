package WRClient

import "wsdk/WRCommon"

type WRClientServer struct {
	*WRCommon.WRServer
	services []*WRCommon.ServiceDescriptor
}

type IWRClientServer interface {
	Services() []*WRCommon.ServiceDescriptor
}
