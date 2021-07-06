package relay_client

import (
	"fmt"
	"net/url"
	"sync"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/roles"
	ws_connection "wsdk/websocket/connection"
	WSClient "wsdk/websocket/wclient"
)

// TODO
type Client struct {
	wclient WSClient.IWClient
	client  roles.ICommonClient
	// serviceMap map[string]IClientService // id -- [listener functions]
	service IClientService
	server  roles.ICommonServer
	lock    *sync.RWMutex
}

func NewClient(serverUri string, serverPort int, myId string) *Client {
	addr := url.URL{Scheme: "ws", Host: fmt.Sprintf("%s:%d", serverUri, serverPort), Path: "/ws"}
	c := &Client{
		wclient: WSClient.New(WSClient.NewWClientConfig(addr.String(), nil, nil, nil, nil, nil)),
		lock:    new(sync.RWMutex),
	}
	c.wclient.SetOnConnectionEstablished(c.onConnected)
	c.wclient.SetOnMessage(func(msg []byte) {
		fmt.Println(msg)
	})
	return c
}

type IWRClient interface {
	Connect() error
	Request(message *messages.Message) (*messages.Message, error)
	Send([]byte) error
	OnMessage(string, func())
	OffMessage(string, func())
	OffAllMessage(string)
}

func (c *Client) Connect() error {
	return c.wclient.Connect()
}

func (c *Client) onConnected(rawConn *ws_connection.WsConnection) {
	// ctx has already started!
	fmt.Println("what's up here!?!?!")
	conn := connection.NewConnection(nil, rawConn, connection.DefaultTimeout, Ctx.MessageParser(), Ctx.NotificationEmitter(), Ctx.TimedJobPool())
	c.client = roles.CreateClient(conn, Ctx.Identity(), "asd", 2)
	err := conn.Send(messages.NewMessage("hello", c.client.Id(), "123", "", messages.MessageTypeACK, ([]byte)("aaa")))
	if err != nil {
		panic(err)
	}
}

func (c *Client) Request(message *messages.Message) (*messages.Message, error) {
	return c.client.Connection().Request(message)
}
