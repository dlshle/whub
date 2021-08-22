package tcp

import (
	"errors"
	"fmt"
	"net"
	"os"
	"wsdk/common/connection"
	"wsdk/common/logger"
)

const DefaultRetryCount = 3

type TCPClient struct {
	serverAddr      string
	serverPort      int
	retryCount      int
	logger          *logger.SimpleLogger
	onConnected     func(conn connection.IConnection)
	onMessage       func([]byte)
	onDisconnected  func(err error)
	onConnectionErr func(err error)

	conn connection.IConnection
}

func NewTCPClient(serverAddr string, serverPort int, myId string) connection.IClient {
	return &TCPClient{
		serverAddr: serverAddr,
		serverPort: serverPort,
		retryCount: DefaultRetryCount,
		logger:     logger.New(os.Stdout, "[TCPClient]", false),
	}
}

func (c *TCPClient) Connect(token string) (connection.IConnection, error) {
	return c.connectWithRetry(c.retryCount, nil)
}

func (c *TCPClient) connectWithRetry(retry int, lastErr error) (connection.IConnection, error) {
	if retry == 0 {
		return nil, lastErr
	}
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", c.serverAddr, c.serverPort))
	if err != nil {
		return c.connectWithRetry(retry-1, err)
	}
	return c.handleConnection(conn)
}

func (c *TCPClient) handleConnection(rawConn net.Conn) (connection.IConnection, error) {
	conn := NewTCPConnection(rawConn)
	conn.OnError(c.onConnectionErr)
	conn.OnClose(c.onDisconnected)
	conn.OnMessage(c.onMessage)
	c.onConnected(conn)
	return conn, nil
}

func (c *TCPClient) ReadLoop() {
	if c.conn != nil {
		c.conn.ReadLoop()
	}
}

func (c *TCPClient) Disconnect() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return errors.New("no connection has been established yet")
}

func (c *TCPClient) Write(data []byte) error {
	if c.conn != nil {
		return c.conn.Write(data)
	}
	return errors.New("no connection has been established yet")
}

func (c *TCPClient) Read() ([]byte, error) {
	if c.conn != nil {
		return c.conn.Read()
	}
	return nil, errors.New("no connection has been established yet")
}

func (c *TCPClient) OnConnectionEstablished(cb func(conn connection.IConnection)) {
	c.onConnected = cb
}

func (c *TCPClient) OnDisconnect(cb func(error)) {
	c.onDisconnected = cb
}

func (c *TCPClient) OnMessage(cb func([]byte)) {
	c.onMessage = cb
}

func (c *TCPClient) OnError(cb func(error)) {
	c.onConnectionErr = cb
}
