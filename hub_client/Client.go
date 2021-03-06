package hub_client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	base_conn "whub/common/connection"
	"whub/common/http"
	"whub/common/logger"
	"whub/hub_client/connections"
	"whub/hub_client/context"
	"whub/hub_common/connection"
	"whub/hub_common/messages"
	"whub/hub_common/roles"
	WSClient "whub/websocket/wclient"
)

// TODO
type Client struct {
	connectionType uint8
	serverUri      string
	loginToken     string
	connPool       connections.IConnectionPool
	wclient        base_conn.IClient
	client         roles.ICommonClient
	httpClient     http.IClientPool
	// serviceMap map[string]IClientService // id -- [listener functions]
	service                     IClientService
	server                      roles.ICommonServer
	primaryConn                 connection.IConnection
	logger                      *logger.SimpleLogger
	lock                        *sync.RWMutex
	dispatcher                  *ClientMessageDispatcher
	clientServiceRequestHandler *ClientServiceMessageHandler
	serviceManager              IServiceManager
}

func NewClient(connType uint8, serverUri string, serverPort int, wsPath string, clientId string, clientCKey string) *Client {
	serverFullUri := fmt.Sprintf("%s:%d", serverUri, serverPort)
	addr := url.URL{Scheme: "ws", Host: serverFullUri, Path: wsPath}
	c := &Client{
		connectionType: connType,
		serverUri:      serverFullUri,
		httpClient:     context.Ctx.HTTPClient(),
		wclient:        WSClient.New(WSClient.NewWClientConfig(addr.String(), nil, nil, nil, nil, nil)),
		client:         roles.NewClient(clientId, "", roles.ClientTypeAnonymous, clientCKey, 0),
		logger:         context.Ctx.Logger(),
		lock:           new(sync.RWMutex),
	}
	c.connPool = connections.NewConnectionPool(c.connect, context.Ctx.MaxActiveServiceConnections()+2)
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
		messages.NewMessage("", "", "", "/auth/login",
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
	err := c.requestAndHandleServerInfo()
	if err != nil {
		return err
	}
	context.Ctx.Start(c.client, c.server)
	c.initDispatchers()
	err = c.Connect()
	if err != nil {
		return err
	}
	c.serviceManager = NewServiceManager(c.primaryConn)
	c.initServiceDispatcher()
	return nil
}

func (c *Client) Connect() error {
	err := c.connPool.Start()
	if err != nil {
		return err
	}
	c.primaryConn, err = c.connPool.Get()
	return err
}

func (c *Client) getServerInfo() (desc roles.RoleDescriptor, err error) {
	resp, err := c.HTTPRequest("", messages.NewMessage("", "", "", "/status/info", messages.MessageTypeServiceRequest, nil))
	if err != nil {
		return
	}
	err = json.Unmarshal(resp.Payload(), &desc)
	return
}

func (c *Client) handleServerInfo(serverInfo roles.RoleDescriptor) error {
	splittedAddr := strings.Split(serverInfo.Address, ":")
	if len(splittedAddr) < 2 {
		return errors.New("invalid server address")
	}
	port, err := strconv.Atoi(splittedAddr[1])
	if err != nil {
		return err
	}
	c.server = roles.NewServer(serverInfo.Id, serverInfo.Description, splittedAddr[0], port)
	return nil
}

func (c *Client) requestAndHandleServerInfo() error {
	serverInfo, err := c.getServerInfo()
	if err != nil {
		return err
	}
	err = c.handleServerInfo(serverInfo)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) connect() (conn connection.IConnection, err error) {
	if c.loginToken == "" {
		c.loginToken, err = c.login(3, nil)
		if err != nil {
			return
		}
	}
	return c.doConnect(3, c.loginToken, nil)
}

func (c *Client) doConnect(retryCount int, token string, lastErr error) (connection.IConnection, error) {
	if retryCount <= 0 {
		return nil, lastErr
	}
	conn, err := c.wclient.Connect(token)
	if err != nil {
		token, err = c.login(2, nil)
		if err != nil {
			return nil, err
		}
		return c.doConnect(retryCount-1, token, err)
	}
	return c.handleConnected(conn)
}

func (c *Client) initDispatchers() {
	c.dispatcher = NewClientMessageDispatcher()
}

func (c *Client) initServiceDispatcher() {
	c.clientServiceRequestHandler = NewClientServiceMessageHandler()
	c.dispatcher.RegisterHandler(c.clientServiceRequestHandler)
}

func (c *Client) handleConnected(rawConn base_conn.IConnection) (connection.IConnection, error) {
	conn := connection.NewConnection(context.Ctx.Logger().WithPrefix("[ServerConnection]"), rawConn, connection.DefaultTimeout, context.Ctx.MessageParser(), context.Ctx.NotificationEmitter())
	c.logger.Println("connection to server has been established: ", conn.Address())
	c.logger.Println("new client has been instantiated")
	context.Ctx.AsyncTaskPool().Schedule(conn.ReadingLoop)
	// test ping
	msg, err := conn.Request(messages.NewPingMessage(c.client.Id(), "123"))
	if err != nil {
		c.logger.Panicln("initial ping error !", err.Error())
		conn.Close()
		return nil, err
	}
	c.logger.Println("test greeting message result: ", msg)
	conn.OnIncomingMessage(func(msg messages.IMessage) {
		c.dispatcher.Dispatch(msg, conn)
	})
	return conn, nil
}

func (c *Client) Request(messageType int, uri string, payload []byte) (messages.IMessage, error) {
	return c.primaryConn.Request(messages.DraftMessage(c.client.Id(), c.server.Id(), uri, messageType, payload))
}

func (c *Client) HTTPRequest(token string, message messages.IMessage) (messages.IMessage, error) {
	r := message.ToHTTPRequest("http", c.serverUri, token)
	resp := c.httpClient.Request(r)
	if resp.Code < 0 || resp.Code > 400 && resp.Code < 510 {
		return nil, errors.New(resp.Body)
	}
	header := resp.Header
	if resp.Header.Get(messages.MessageHTTPHeaderId) != "" {
		return messages.NewMessage(header.Get(messages.MessageHTTPHeaderId), header.Get(messages.MessageHTTPHeaderFrom), header.Get(messages.MessageHTTPHeaderTo),
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

func (c *Client) RegisterService(service IClientService) error {
	return c.serviceManager.RegisterService(service)
}

func (c *Client) StartService(id string) error {
	svc := c.serviceManager.GetServiceById(id)
	if svc == nil {
		return errors.New(fmt.Sprintf("service %s has not been registered yet", id))
	}
	return svc.Start()
}

func (c *Client) StopService(id string) error {
	svc := c.serviceManager.GetServiceById(id)
	if svc == nil {
		return errors.New(fmt.Sprintf("service %s has not been registered yet", id))
	}
	return svc.Stop()
}

func (c *Client) Stop() {
	c.connPool.Close()
	c.serviceManager.UnregisterAllServices()
}
