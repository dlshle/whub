package relay_client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	base_conn "wsdk/common/connection"
	"wsdk/common/http"
	"wsdk/common/logger"
	"wsdk/relay_client/connections"
	"wsdk/relay_client/context"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/roles"
	WSClient "wsdk/websocket/wclient"
)

// TODO
type Client struct {
	connectionType uint8
	serverUri      string
	loginToken     string
	connPool       connections.IConnectionPool
	wclient        base_conn.IClient
	client         roles.ICommonClient
	httpClient     *http.ClientPool
	// serviceMap map[string]IClientService // id -- [listener functions]
	service                     IClientService
	server                      roles.ICommonServer
	conn                        connection.IConnection
	logger                      *logger.SimpleLogger
	lock                        *sync.RWMutex
	dispatcher                  *ClientMessageDispatcher
	clientServiceRequestHandler *ClientServiceMessageHandler
}

func NewClient(connType uint8, serverUri string, serverPort int, wsPath string, clientId string, clientCKey string) *Client {
	serverFullUri := fmt.Sprintf("%s:%d", serverUri, serverPort)
	addr := url.URL{Scheme: "ws", Host: serverFullUri, Path: connection.WSConnectionPath}
	c := &Client{
		connectionType: connType,
		serverUri:      serverFullUri,
		httpClient:     http.NewPool(clientId, 5, 128, 60),
		wclient:        WSClient.New(WSClient.NewWClientConfig(addr.String(), nil, nil, nil, nil, nil)),
		client:         roles.NewClient(clientId, "", roles.ClientTypeAnonymous, clientCKey, 0),
		logger:         context.Ctx.Logger(),
		lock:           new(sync.RWMutex),
	}
	c.connPool = connections.NewConnectionPool(c.connect, 5)
	// TODO no good, it will make connect stuck there forever or sth else?
	c.wclient.OnConnectionEstablished(func(rawConn base_conn.IConnection) {
		c.handleConnected(rawConn)
	})
	c.wclient.OnMessage(func(msg []byte) {
		fmt.Println(msg)
	})
	return c
}

type IWRClient interface {
	Connect() error
	Request(message messages.IMessage) (messages.IMessage, error)
	Send([]byte) error
	OnMessage(string, func())
	OffMessage(string, func())
	OffAllMessage(string)
	Role() roles.ICommonClient
}

type LoginResp struct {
	Token string `json:"token"`
}

func (c *Client) login(retry int, err error) (string, error) {
	if retry <= 0 {
		return "", err
	}
	loginBody := ([]byte)(fmt.Sprintf("{\"id\":\"%s\",\"password\":\"%s\"}", c.client.Id(), c.client.CKey()))
	resp, err := c.HTTPRequest("",
		messages.NewMessage("", "", "", "/service/client/login",
			messages.MessageTypeServicePostRequest, loginBody))
	if err != nil {
		return c.login(retry-1, err)
	}
	var loginResp LoginResp
	err = json.Unmarshal(resp.Payload(), &loginResp)
	if err != nil {
		return "", err
	}
	return loginResp.Token, nil
}

func (c *Client) Start() error {
	// err := c.connPool.Start()
	// if err != nil {
	// return err
	// }
	// add other connections
	// <-context.Ctx.Context().Done()
	return c.Connect()
}

func (c *Client) Connect() error {
	token, err := c.login(5, nil)
	if err != nil {
		return err
	}
	_, err = c.wclient.Connect(token)
	return err
}

func (c *Client) connect() (connection.IConnection, error) {
	token, err := c.login(3, nil)
	if err != nil {
		return nil, err
	}
	return c.doConnect(3, token, nil)
}

func (c *Client) doConnect(retryCount int, token string, lastErr error) (connection.IConnection, error) {
	if retryCount <= 0 {
		return nil, lastErr
	}
	conn, err := c.wclient.Connect(token)
	if err != nil {
		token, err = c.login(1, nil)
		if err != nil {
			return nil, err
		}
		return c.doConnect(retryCount-1, token, err)
	}
	return c.handleConnected(conn), err
}

func (c *Client) initDispatchers() {
	c.dispatcher = NewClientMessageDispatcher()
	c.clientServiceRequestHandler = NewClientServiceMessageHandler()
	c.dispatcher.RegisterHandler(c.clientServiceRequestHandler)
}

func (c *Client) handleConnected(rawConn base_conn.IConnection) connection.IConnection {
	// ctx has already started!
	conn := connection.NewConnection(context.Ctx.Logger().WithPrefix("[ServerConnection]"), rawConn, connection.DefaultTimeout, context.Ctx.MessageParser(), context.Ctx.NotificationEmitter())
	c.conn = conn
	c.logger.Println("connection to server has been established: ", conn.Address())
	c.client = roles.NewClient("aa", "bb", roles.RoleTypeClient, "asd", 2)
	c.logger.Println("new client has been instantiated")
	context.Ctx.AsyncTaskPool().Schedule(c.conn.ReadingLoop)
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
	context.Ctx.Start(c.client, c.server)
	c.initDispatchers()
	err = conn.Send(messages.NewMessage("hello", c.client.Id(), "123", "", messages.MessageTypeACK, ([]byte)("aaa")))
	c.logger.Println("greeting message has been sent")
	conn.OnIncomingMessage(func(msg messages.IMessage) {
		// TODO remove later
		c.logger.Printf("client conn %p received message", conn)
		c.dispatcher.Dispatch(msg, conn)
	})
	if err != nil {
		c.logger.Panicln("greeting error !", err.Error())
		panic(err)
	}
	return conn
}

func (c *Client) Request(message messages.IMessage) (messages.IMessage, error) {
	return c.conn.Request(message)
	//	conn, err := c.connPool.Get()
	//	if err != nil {
	//		return nil, err
	//	}
	//	return conn.Request(message)
}

func (c *Client) HTTPRequest(token string, message messages.IMessage) (messages.IMessage, error) {
	r := message.ToHTTPRequest("http", c.serverUri, token)
	resp := c.httpClient.Request(r)
	if resp.Code < 0 {
		return nil, errors.New(resp.Body)
	}
	header := resp.Header
	if resp.Header.Get("Message-Id") != "" {
		return messages.NewMessage(header.Get("Message-Id"), header.Get("From"), header.Get("To"),
			message.Uri(), resp.Code, ([]byte)(resp.Body)), nil
	}
	return nil, errors.New("invalid server response")
}

func (c *Client) httpRequest(r *http.Request) (messages.IMessage, error) {
	resp := c.httpClient.Request(r)
	if resp.Code < 0 {
		return nil, errors.New(resp.Body)
	}
	header := resp.Header
	if resp.Header.Get("Message-Id") != "" {
		return messages.NewMessage(header.Get("Message-Id"), header.Get("From"), header.Get("To"),
			// HOW TO GET URI?
			"", resp.Code, ([]byte)(resp.Body)), nil
	}
	return nil, errors.New("invalid server response")
}

func (c *Client) Role() roles.ICommonClient {
	return c.client
}

func (c *Client) SetService(service IClientService) {
	c.service = service
	c.clientServiceRequestHandler.SetService(service)
}

func (c *Client) RegisterService() (err error) {
	if c.service != nil {
		err = c.service.Init(c.server, c.conn)
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
