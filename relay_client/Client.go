package relay_client

import (
	"fmt"
	"net/url"
	"sync"
	"time"
	"wsdk/common/logger"
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
	logger  *logger.SimpleLogger
	lock    *sync.RWMutex
}

func NewClient(serverUri string, serverPort int, myId string) *Client {
	addr := url.URL{Scheme: "ws", Host: fmt.Sprintf("%s:%d", serverUri, serverPort), Path: "/ws"}
	c := &Client{
		wclient: WSClient.New(WSClient.NewWClientConfig(addr.String(), nil, nil, nil, nil, nil)),
		logger:  Ctx.Logger(),
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
	conn := connection.NewConnection(Ctx.Logger().WithPrefix("[ServerConnection]"), rawConn, connection.DefaultTimeout, Ctx.MessageParser(), Ctx.NotificationEmitter())
	c.logger.Println("connection to server has been established: ", conn.Address())
	c.client = roles.NewClient(conn, "aa", "bb", roles.RoleTypeClient, "asd", 2)
	c.logger.Println("new client has been instantiated")
	err := conn.Send(messages.NewMessage("hello", c.client.Id(), "123", "", messages.MessageTypeACK, ([]byte)("aaa")))
	c.logger.Println("greeting message has been sent")
	conn.OnIncomingMessage(func(msg *messages.Message) {
		conn.Send(messages.NewMessage(msg.Id(), conn.Address(), msg.From(), msg.Uri(), messages.MessageTypeACK, nil))
	})
	if err != nil {
		c.logger.Panicln("greeting error !", err.Error())
		panic(err)
	}
	time.Sleep(time.Minute * 1)
}

func (c *Client) Request(message *messages.Message) (*messages.Message, error) {
	return c.client.Connection().Request(message)
}
