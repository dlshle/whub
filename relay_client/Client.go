package relay_client

import (
	"sync"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/roles"
	ws_connection "wsdk/websocket/connection"
	WSClient "wsdk/websocket/wclient"
)

// TODO
type Client struct {
	wclient *WSClient.WClient
	client roles.ICommonClient
	// serviceMap map[string]IClientService // id -- [listener functions]
	service IClientService
	server  roles.ICommonServer
	lock    *sync.RWMutex
}

type IWRClient interface {
	Request(message *messages.Message) (*messages.Message, error)
	Send([]byte) error
	OnMessage(string, func())
	OffMessage(string, func())
	OffAllMessage(string)
}

func (c *Client) onConnected(rawConn *ws_connection.WsConnection) error {
	// ctx has already started!
	conn := connection.NewConnection(rawConn, connection.DefaultTimeout, Ctx.MessageParser(), Ctx.NotificationEmitter()))
	c.client = roles.CreateClient(conn, Ctx.Identity(), "asd",2)
}

func (c *Client) Request(message *messages.Message) (*messages.Message, error) {
	return c.client.Connection().Request(message)
}