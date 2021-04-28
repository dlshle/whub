package WRelayServer

import (
	"wsdk/WRCommon"
	"wsdk/WServer"
)

type WRelayServer struct {
	*WServer.WServer
	*WRCommon.WRBaseRole
	clients map[string]*WRCommon.WRClient
}

func New(id string, description string, ip string, port int) *WRelayServer {

}