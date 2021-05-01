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

/* General health check strategy: client should send PING to server every X seconds, and if server does not receive a
   in X + 1 seconds, server will send PING and expect to have a PONG received in X seconds. If that fails, health check
   is considered failed.
*/

/*
func New(id string, description string, ip string, port int) *WRelayServer {

}
*/
