package WRelayServer

import (
	"github.com/dlshle/gommon/timed"
	"time"
	"wsdk/WRCommon"
	"wsdk/WServer"
)

type WRelayServer struct {
	*WServer.WServer
	*WRCommon.WRBaseRole
	clients map[string]*WRCommon.WRClient
	serviceMap map[string]IService // client-id <--> Service when a client is closed, should also kill the service until it's expired(Tdead + Texipre_period)
	serviceExpirePeriod time.Duration
	scheduleJobPool timed.IJobPool
}

// server *-- services
// server *-- clients
// service *-- client
// when server knows the client is disconnected, server should put service in survival mode(constantly health check with client id until client is recovered or service expired)

/* General health check strategy: client should send PING to server every X seconds, and if server does not receive a
   in X + 1 seconds, server will send PING and expect to have a PONG received in X seconds. If that fails, health check
   is considered failed.
*/

/*
func New(id string, description string, ip string, port int) *WRelayServer {

}
*/
