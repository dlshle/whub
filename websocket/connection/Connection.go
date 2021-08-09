package connection

import (
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"sync"
	"time"
	"wsdk/common/connection"
)

const (
	StateIdle         = 0
	StateReading      = 1
	StateStopping     = 2
	StateStopped      = 3
	StateClosing      = 4
	StateDisconnected = 5
)

type WsConnection struct {
	conn          *websocket.Conn
	onMessage     func([]byte)
	onClose       func(error)
	onError       func(error)
	connectedTime time.Time
	lastRecvTime  time.Time
	lastSendTime  time.Time
	state         int
	lock          *sync.Mutex
	closeChannel  chan bool
}

func NewWsConnection(conn *websocket.Conn, onMessage func([]byte), onClose func(error), onError func(error)) connection.IConnection {
	now := time.Now()
	return &WsConnection{conn, onMessage, onClose, onError, now, now, now, 0, new(sync.Mutex), make(chan bool)}
}

type IWsConnection interface {
	Close() error
	Read() ([]byte, error)
	OnMessage(func([]byte))
	Write([]byte) error
	Address() string
	OnError(func(error))
	OnClose(func(error))
	State() int
	ReadLoop()
	String() string
}

func (c *WsConnection) withLock(cb func()) {
	c.lock.Lock()
	defer c.lock.Unlock()
	cb()
}

func (c *WsConnection) State() int {
	return c.state
}

func (c *WsConnection) setState(state int) {
	if state < 0 || state > StateDisconnected {
		return
	}
	c.state = state
}

func (c *WsConnection) Close() (err error) {
	if c.State() >= StateClosing {
		return errors.New("err: closing a closing connection")
	}
	c.setState(StateClosing)
	err = c.conn.Close()
	if c.onClose != nil {
		c.onClose(err)
	}
	c.setState(StateDisconnected)
	<-c.closeChannel
	return err
}

func (c *WsConnection) ReadLoop() {
	if c.State() > StateIdle {
		return
	}
	c.setState(StateReading)
	// c.conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
	for c.State() == StateReading {
		// Read will handle error itself
		msg, err := c.Read()
		if err == nil && c.onMessage != nil {
			c.onMessage(msg)
		} else if err != nil {
			break
		}
	}
	c.setState(StateStopped)
}

func (c *WsConnection) Read() ([]byte, error) {
	_, stream, err := c.conn.ReadMessage()
	if err != nil {
		c.handleError(err)
	} else if err == nil {
		c.lastRecvTime = time.Now()
	}
	return stream, err
}

func (c *WsConnection) Write(stream []byte) (err error) {
	// use write lock to prevent concurrent write
	c.withLock(func() {
		t := time.Now()
		err = c.conn.WriteMessage(1, stream)
		if err != nil {
			c.lastSendTime = t
		}
	})
	return
}

func (c *WsConnection) Address() string {
	return c.conn.RemoteAddr().String()
}

func (c *WsConnection) OnClose(cb func(error)) {
	c.onClose = cb
}

func (c *WsConnection) OnError(cb func(error)) {
	c.onError = cb
}

func (c *WsConnection) OnMessage(cb func([]byte)) {
	c.onMessage = cb
}

func (c *WsConnection) String() string {
	return fmt.Sprintf("{\"address\": \"%s\",\"state\": %d }", c.Address(), c.State())
}

func (c *WsConnection) handleError(err error) {
	if c.onError == nil {
		c.Close()
	} else {
		c.onError(err)
	}
}

func (c *WsConnection) ConnectionType() uint8 {
	return connection.TypeWS
}

func (c *WsConnection) IsLive() bool {
	return c.State() == StateReading
}
