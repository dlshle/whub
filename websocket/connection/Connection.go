package connection

import (
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"sync"
	"time"
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
	isClosing     bool
	state         int
	rwLock        *sync.RWMutex
	closeChannel  chan bool
}

func NewWsConnection(conn *websocket.Conn, onMessage func([]byte), onClose func(error), onError func(error)) *WsConnection {
	now := time.Now()
	return &WsConnection{conn, onMessage, onClose, onError, now, now, now, false, 0, new(sync.RWMutex), make(chan bool)}
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
	StartListening()
	StopListening()
	String() string
}

func (c *WsConnection) withWrite(cb func()) {
	c.rwLock.Lock()
	defer c.rwLock.Unlock()
	cb()
}

func (c *WsConnection) withRead(cb func() interface{}) interface{} {
	c.rwLock.RLock()
	defer c.rwLock.RUnlock()
	return cb()
}

func (c *WsConnection) State() int {
	return c.withRead(func() interface{} {
		return c.state
	}).(int)
}

func (c *WsConnection) setState(state int) {
	if state < 0 || state > StateDisconnected {
		return
	}
	c.withWrite(func() {
		c.state = state
	})
}

func (c *WsConnection) Close() (err error) {
	if c.State() >= StateClosing {
		return errors.New("err: closing a closing connection")
	}
	c.setState(StateClosing)
	c.rwLock.Lock()
	c.isClosing = true
	c.rwLock.Unlock()
	err = c.conn.Close()
	if c.onClose != nil {
		c.onClose(err)
	}
	c.setState(StateDisconnected)
	<-c.closeChannel
	return err
}

func (c *WsConnection) StartListening() {
	if c.State() > StateIdle {
		return
	}
	c.setState(StateReading)
	go func() {
		// c.conn.SetWriteDeadline(time.Now().Schedule(30 * time.Second))
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
		close(c.closeChannel)
	}()
}

func (c *WsConnection) StopListening() {
	if c.State() != StateReading {
		return
	}
	c.setState(StateStopping)
	<-c.closeChannel
}

func (c *WsConnection) Read() ([]byte, error) {
	_, stream, err := c.conn.ReadMessage()
	if err != nil && c.onError != nil {
		c.onError(err)
	} else if err == nil {
		c.lastRecvTime = time.Now()
	}
	return stream, err
}

func (c *WsConnection) Write(stream []byte) (err error) {
	t := time.Now()
	err = c.conn.WriteMessage(1, stream)
	if err != nil {
		c.lastSendTime = t
	}
	return
}

func (c *WsConnection) Address() string {
	return c.conn.RemoteAddr().String()
}

func (c *WsConnection) OnClose(cb func(error)) {
	c.withWrite(func() {
		c.onClose = cb
	})
}

func (c *WsConnection) OnError(cb func(error)) {
	c.withWrite(func() {
		c.onError = cb
	})
}

func (c *WsConnection) OnMessage(cb func([]byte)) {
	c.withWrite(func() {
		c.onMessage = cb
	})
}

func (c *WsConnection) String() string {
	return fmt.Sprintf("WsConnection { address: %s, state: %d }", c.Address(), c.State())
}
