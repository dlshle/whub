package relay_client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
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
	service                     IClientService
	server                      roles.ICommonServer
	conn                        connection.IConnection
	logger                      *logger.SimpleLogger
	lock                        *sync.RWMutex
	dispatcher                  *ClientMessageDispatcher
	clientServiceRequestHandler *ClientServiceMessageHandler
}

func NewClient(serverUri string, serverPort int, myId string) *Client {
	addr := url.URL{Scheme: "ws", Host: fmt.Sprintf("%s:%d", serverUri, serverPort), Path: connection.WSConnectionPath}
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
	Role() roles.ICommonClient
}

func (c *Client) Connect() error {
	err := c.wclient.Connect()
	return err
}

func (c *Client) initDispatchers() {
	c.dispatcher = NewClientMessageDispatcher()
	c.clientServiceRequestHandler = NewClientServiceMessageHandler()
	c.dispatcher.RegisterHandler(c.clientServiceRequestHandler)
}

func (c *Client) onConnected(rawConn *ws_connection.WsConnection) {
	// ctx has already started!
	conn := connection.NewConnection(Ctx.Logger().WithPrefix("[ServerConnection]"), rawConn, connection.DefaultTimeout, Ctx.MessageParser(), Ctx.NotificationEmitter())
	c.conn = conn
	c.logger.Println("connection to server has been established: ", conn.Address())
	c.client = roles.NewClient(conn, "aa", "bb", roles.RoleTypeClient, "asd", 2)
	c.logger.Println("new client has been instantiated")
	c.wclient.ListenToMessage()
	// TODO should request ClientDescriptor to the server
	serverDesc, err := conn.Request(messages.DraftMessage(c.client.Id(), "", "", messages.MessageTypeClientDescriptor, ([]byte)(c.client.Describe().String())))
	if err != nil {
		c.logger.Println("unable to receive server description due to ", err.Error())
		panic(err)
	}
	c.logger.Println("receive server description response: ", serverDesc)
	// TODO refactor with better coding fuck
	var serverRoleDesc roles.RoleDescriptor
	err = json.Unmarshal(serverDesc.Payload(), &serverRoleDesc)
	if err != nil {
		c.logger.Println("unable to unmarshall server role descriptor due to ", err.Error())
		panic(err)
	}
	splittedAddr := strings.Split(serverRoleDesc.Address, ":")
	port, err := strconv.Atoi(splittedAddr[1])
	if err != nil {
		c.logger.Println("unable to parse server port ", err.Error())
		panic(err)
	}
	c.server = roles.NewServer(serverRoleDesc.Id, serverRoleDesc.Description, splittedAddr[0], port)
	Ctx.Start(c.client, c.server)
	c.initDispatchers()
	err = conn.Send(messages.NewMessage("hello", c.client.Id(), "123", "", messages.MessageTypeACK, ([]byte)("aaa")))
	c.logger.Println("greeting message has been sent")
	conn.OnIncomingMessage(func(msg *messages.Message) {
		c.dispatcher.Dispatch(msg, conn)
	})
	if err != nil {
		c.logger.Panicln("greeting error !", err.Error())
		panic(err)
	}
}

func (c *Client) Request(message *messages.Message) (*messages.Message, error) {
	return c.client.Connection().Request(message)
}

func (c *Client) Role() roles.ICommonClient {
	return c.client
}

func (c *Client) SetService(service IClientService) {
	c.service = service
	c.clientServiceRequestHandler.SetService(service)
}

func (c *Client) RegisterService() error {
	if c.service != nil {
		err := c.service.Init(c.server, c.conn)
		if err != nil {
			c.logger.Println("Init service failed due to ", err.Error())
			return err
		}
		return c.service.Register()
	}
	return errors.New("no service is present")
}

func (c *Client) StartService() error {
	if c.service != nil {
		return c.service.Start()
	}
	return errors.New("no service is present")
}

func (c *Client) StopService() error {
	if c.service != nil {
		return c.service.Stop()
	}
	return errors.New("no service is present")
}
