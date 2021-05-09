package relay_server

import (
	"encoding/json"
	"github.com/dlshle/gommon/timed"
	"sync"
	"time"
	"wsdk/base/common"
	"wsdk/base/wserver"
	"wsdk/relay_common"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/utils"
)

type WRelayServer struct {
	ctx relay_common.IWRContext
	*wserver.WServer
	relay_common.IDescribableRole
	anonymousClient map[string]*WRServerClient // raw clients or pure anony clients
	clients map[string]*WRServerClient
	serviceMap map[string]IServerService // client-id <--> ServerService when a client is closed, should also kill the service until it's expired(Tdead + Texipre_period)
	serviceExpirePeriod time.Duration
	scheduleJobPool *timed.JobPool
	messageHandler messages.IMessageHandler
	lock *sync.RWMutex
}

type clientExtraInfoDescriptor struct {
	pScope int
	cKey string
	cType int
}

func (s *WRelayServer) withWrite(cb func()) {
	s.lock.Lock()
	defer s.lock.Unlock()
	cb()
}

func (s *WRelayServer) handleInitialConnection(conn *common.WsConnection) {
	rawClient := NewAnonymousClient(connection.NewWRConnection(conn, connection.DefaultTimeout, s.messageHandler, s.ctx.NotificationEmitter()))
	s.withWrite(func() {
		s.anonymousClient[conn.Address()] = rawClient
	})
	resp, err := rawClient.Request(rawClient.NewMessage(s.Id(), messages.MessageTypeServerDescriptor, ([]byte)(s.Describe().String())))
	if err == nil && resp.MessageType() == messages.MessageTypeClientDescriptor {
		var clientDescriptor relay_common.RoleDescriptor
		var clientExtraInfo clientExtraInfoDescriptor
		err = utils.ProcessWithError([]func()error{
			func() error {
				return json.Unmarshal(resp.Payload(), &clientDescriptor)
			},
			func() error {
				return json.Unmarshal(([]byte)(clientDescriptor.ExtraInfo), &clientExtraInfo)
			},
		})
		if err != nil {
			s.withWrite(func() {
				delete(s.anonymousClient, conn.Address())
				s.clients[clientDescriptor.Id] = NewClient(rawClient.WRConnection, clientDescriptor.Id, clientDescriptor.Description, clientExtraInfo.cType, clientExtraInfo.cKey, clientExtraInfo.pScope)
			})
		}
	}
}

func NewServer(ctx relay_common.IWRContext, id string, description string, port int) *WRelayServer {
	server := &WRelayServer{
		ctx: ctx,
		WServer: wserver.NewWServer(wserver.NewServerConfig(id, "127.0.0.1", port, wserver.DefaultWsConnHandler())),
		IDescribableRole: ctx.Identity(),
		anonymousClient: make(map[string]*WRServerClient),
		clients: make(map[string]*WRServerClient),
		serviceMap: make(map[string]IServerService),
		serviceExpirePeriod: time.Second,
		scheduleJobPool: ctx.TimedJobPool(),
		messageHandler: messages.NewSimpleMessageHandler(),
		lock: new(sync.RWMutex),
	}
	server.OnClientConnected(server.handleInitialConnection)
	// TODO
	/*
		onClientClosed func(conn *common.WsConnection, err error),
		onHttpRequest func(u func(w http.ResponseWriter, r *http.Request) error, w http.ResponseWriter, r *http.Request),
		onConnectionError func(*common.WsConnection, error)
	 */
	return server
}

// TODO
// server *-- services
// server *-- clients
// service *-- client
// when server knows the client is disconnected, server should put service in survival mode(constantly health check with client id until client is recovered or service expired)

/* General health check strategy: client should send PING to server every X seconds, and if server does not receive a
   in X + 1 seconds, server will send PING and expect to have a PONG received in X seconds. If that fails, health check
   is considered failed.
*/

/*
func New(id string, description string, ip string, port int) *relay_server {

}
*/
