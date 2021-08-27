package WSClient

import (
	"fmt"
	"github.com/gorilla/websocket"
	"os"
	base_conn "wsdk/common/connection"
	"wsdk/common/logger"
	"wsdk/websocket/connection"
)

type WClientConnectionHandler struct {
	onMessage               func([]byte)
	onConnectionEstablished func(base_conn.IConnection)
	onConnectionFailed      func(error)
	onDisconnected          func(error)
	onError                 func(error)
}

func (c *WClientConnectionHandler) OnMessage(msg []byte) {
	if c.onMessage != nil {
		c.onMessage(msg)
	}
}

func (c *WClientConnectionHandler) OnConnectionEstablished(connection base_conn.IConnection) {
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
	OnConnectionEstablished(*connection.WsConnection)
	OnConnectionFailed(error)
	OnDisconnected(error)
	OnError(error)
}

type WClientConfig struct {
	*WClientConnectionHandler
	serverUrl string
}

func NewWClientConfig(serverUrl string, onMessage func([]byte), onConnectionEstablished func(connection base_conn.IConnection), onConnectionFailed func(error), onDisconnected func(error), onError func(error)) *WClientConfig {
	return &WClientConfig{&WClientConnectionHandler{onMessage, onConnectionEstablished, onConnectionFailed, onDisconnected, onError}, serverUrl}
}

type WClient struct {
	serverUrl string
	handler   *WClientConnectionHandler
	logger    *logger.SimpleLogger
	// conn      base_conn.IConnection
}

func New(config *WClientConfig) base_conn.IClient {
	return &WClient{config.serverUrl, config.WClientConnectionHandler, logger.New(os.Stdout, "[WebSocketClient]", true)}
}

func (c *WClient) Connect(token string) (base_conn.IConnection, error) {
	header := make(map[string][]string)
	header["Authorization"] = []string{fmt.Sprintf("Bearer %s", token)}
	conn, _, err := websocket.DefaultDialer.Dial(c.serverUrl, header)
	if err != nil {
		c.handler.OnConnectionFailed(err)
		return nil, err
	}
	connection := connection.NewWsConnection(conn, c.handler.onMessage, c.handler.onDisconnected, c.handler.onError)
	// c.conn = connection
	c.handler.OnConnectionEstablished(connection)
	return connection, nil
}

func (c *WClient) checkConn() error {
	//	if c.conn == nil {
	//		return errors.New("connection is not established yet")
	//	}
	return nil
}

func (c *WClient) Disconnect() error {
	err := c.checkConn()
	if err != nil {
		return err
	}
	// return c.conn.Close()
	return nil
}

/*
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

func (c *WClient) ReadLoop() {
	c.conn.ReadLoop()
}
*/

func (c *WClient) OnDisconnect(cb func(error)) {
	c.handler.onDisconnected = cb
}

func (c *WClient) OnMessage(cb func([]byte)) {
	c.handler.onMessage = cb
}

func (c *WClient) OnError(cb func(error)) {
	c.handler.onError = cb
}

func (c *WClient) OnConnectionEstablished(cb func(conn base_conn.IConnection)) {
	c.handler.onConnectionEstablished = cb
}
