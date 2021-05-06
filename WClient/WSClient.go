package WSClient

import (
	"errors"
	"github.com/dlshle/gommon/logger"
	"github.com/gorilla/websocket"
	"os"
	"wsdk/Common"
)

type WClientConnectionHandler struct {
	onMessage func([]byte)
	onConnectionEstablished func(*Common.WsConnection)
	onConnectionFailed func(error)
	onDisconnected func(error)
	onError func(error)
}

func (c *WClientConnectionHandler) OnMessage(msg []byte) {
	if c.onMessage != nil {
		c.onMessage(msg)
	}
}

func (c *WClientConnectionHandler) OnConnectionEstablished(connection *Common.WsConnection) {
	if c.onConnectionEstablished != nil {
		c.onConnectionEstablished(connection)
	}
}

func (c *WClientConnectionHandler) OnConnectionFailed(err error) {
	if c.onConnectionFailed != nil {
		c.onConnectionFailed(err)
	}
}

func (c *WClientConnectionHandler) OnDisconnected(err error) {
	if c.onDisconnected != nil {
		c.onDisconnected(err)
	}
}

func (c *WClientConnectionHandler) OnError(err error) {
	if c.onError != nil {
		c.onError(err)
	}
}

type IWClientConnectionHandler interface {
	OnConnectionEstablished(*Common.WsConnection)
	OnConnectionFailed(error)
	OnDisconnected(error)
	OnError(error)
}

type WClientConfig struct {
	*WClientConnectionHandler
	serverUrl string
}

func NewWClientConfig(serverUrl string, onMessage func([]byte), onConnectionEstablished func(connection *Common.WsConnection), onConnectionFailed func(error), onDisconnected func(error), onError func(error)) *WClientConfig {
	return &WClientConfig{&WClientConnectionHandler{onMessage, onConnectionEstablished, onConnectionFailed, onDisconnected, onError}, serverUrl}
}

type WClient struct {
	serverUrl string
	handler *WClientConnectionHandler
	logger    *logger.SimpleLogger
	conn      *Common.WsConnection
}

func New(config *WClientConfig) *WClient {
	return &WClient{config.serverUrl, config.WClientConnectionHandler, logger.New(os.Stdout, "[WClient]", true), nil}
}

func NewClient(serverUrl string) *WClient {
	return New(NewWClientConfig(serverUrl, nil, nil, nil,nil,nil))
}

type IWClient interface {
	Connect() error
	Disconnect() error
	Write(data []byte) error
	Read() ([]byte, error)
	SetOnDisconnect(func(error))
	SetOnMessage(func([]byte))
	SetOnError(func(error))
	ListenToMessage()
	StopListenToMessage()
}

func (c *WClient) Connect() error {
	conn, _, err := websocket.DefaultDialer.Dial(c.serverUrl, nil)
	if err != nil {
		c.handler.OnConnectionFailed(err)
		return err
	}
	connection := Common.NewWsConnection(conn, func(msg []byte) {
		c.handler.OnMessage(msg)
	}, func(err error) {
		c.handler.OnDisconnected(err)
	}, func(err error) {
		c.handler.OnError(err)
	})
	c.conn = connection
	c.handler.OnConnectionEstablished(connection)
	return nil
}

func (c *WClient) checkConn() error {
	if c.conn == nil {
		return errors.New("connection is not established yet")
	}
	return nil
}

func (c *WClient) Disconnect() error {
	err := c.checkConn()
	if err != nil {
		return err
	}
	return c.conn.Close()
}

func (c *WClient) Write(data []byte) error {
	err := c.checkConn()
	if err != nil {
		return err
	}
	return c.conn.Write(data)
}

// Should deprecate this one!
func (c *WClient) Read() ([]byte, error) {
	return c.conn.Read()
}

func (c *WClient) OnDisconnected(cb func(error)) {
	c.handler.onDisconnected = cb
}

func (c *WClient) OnMessage(cb func([]byte)) {
	c.handler.onMessage = cb
}

func (c *WClient) OnError(cb func(error)) {
	c.handler.onError = cb
}

// TODO not so good...
func (c *WClient) ListenToMessage() {
	if c.conn != nil {
		c.conn.StartListening()
	}
}

func (c *WClient) StopListenToMessage() {
	if c.conn != nil {
		c.conn.StopListening()
	}
}
