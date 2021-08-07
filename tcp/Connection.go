package tcp

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"wsdk/common/connection"
)

const (
	DefaultReadBufferSize = 4096
)

type TCPConnection struct {
	conn        net.Conn
	onMessageCb func([]byte)
	onCloseCb   func(error)
	onErrorCb   func(error)
	state       int

	rwLock    *sync.RWMutex
	closeChan chan bool
}

func NewTCPConnection(conn net.Conn) connection.IConnection {
	return &TCPConnection{
		conn:      conn,
		state:     connection.StateIdle,
		rwLock:    new(sync.RWMutex),
		closeChan: make(chan bool),
	}
}

func (c *TCPConnection) withWrite(cb func()) {
	c.rwLock.RLock()
	defer c.rwLock.RUnlock()
	cb()
}

func (c *TCPConnection) setState(state int) {
	if state > 0 && state <= connection.StateDisconnected {
		c.withWrite(func() {
			c.state = state
		})
	}
}

func (c *TCPConnection) Close() error {
	if c.State() >= connection.StateClosing {
		return errors.New("err: closing a closing connection")
	}
	c.setState(connection.StateClosing)
	err := c.conn.Close()
	c.handleClose(err)
	c.setState(connection.StateDisconnected)
	return err
}

func (c *TCPConnection) Read() ([]byte, error) {
	buffer := make([]byte, DefaultReadBufferSize)
	_, err := c.conn.Read(buffer)
	if err != nil {
		c.handleError(err)
	} else {
		c.handleMessage(buffer)
	}
	return buffer, err
}

func (c *TCPConnection) OnMessage(cb func([]byte)) {
	c.onMessageCb = cb
}

func (c *TCPConnection) handleMessage(message []byte) {
	if c.onMessageCb != nil {
		c.onMessageCb(message)
	}
}

func (c *TCPConnection) Write(data []byte) (err error) {
	c.withWrite(func() {
		_, err = c.conn.Write(data)
	})
	if err != nil {
		c.handleError(err)
	}
	return err
}

func (c *TCPConnection) Address() string {
	return c.conn.RemoteAddr().String()
}

func (c *TCPConnection) OnError(cb func(err error)) {
	c.onErrorCb = cb
}

func (c *TCPConnection) handleError(err error) {
	if c.onErrorCb != nil {
		c.onErrorCb(err)
	} else {
		c.Close()
	}
}

func (c *TCPConnection) OnClose(cb func(err error)) {
	c.onCloseCb = cb
}

func (c *TCPConnection) handleClose(err error) {
	if c.onCloseCb != nil {
		c.onCloseCb(err)
	}
}

func (c *TCPConnection) State() int {
	c.rwLock.RLock()
	defer c.rwLock.RUnlock()
	return c.state
}

func (c *TCPConnection) ReadLoop() {
	if c.State() > connection.StateIdle {
		return
	}
	c.setState(connection.StateReading)
	// c.conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
	for c.State() == connection.StateReading {
		// Read will handle error itself
		msg, err := c.Read()
		if err == nil {
			c.handleMessage(msg)
		} else if err != nil {
			break
		}
	}
	c.setState(connection.StateStopped)
	close(c.closeChan)
}

func (c *TCPConnection) String() string {
	return fmt.Sprintf("{\"address\": \"%s\",\"state\": %d }", c.Address(), c.State())
}

func (c *TCPConnection) ConnectionType() uint8 {
	return connection.TypeTCP
}
