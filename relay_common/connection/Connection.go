package connection

import (
	"errors"
	"time"
	"wsdk/common/async"
	"wsdk/common/timed"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/notification"
	"wsdk/websocket/connection"
)

const DefaultTimeout = time.Second * 30

type Connection struct {
	ws                  connection.IWsConnection
	requestTimeout      time.Duration
	messageParser       messages.IMessageParser
	notificationEmitter notification.IWRNotificationEmitter
	messageCallback     func(*messages.Message)
}

type IConnection interface {
	Address() string
	AsyncRequest(*messages.Message) (*async.StatefulBarrier, error)
	Request(*messages.Message) (*messages.Message, error)
	RequestWithTimeout(*messages.Message, time.Duration) (*messages.Message, error)
	Send(*messages.Message) error
	OnIncomingMessage(func(message *messages.Message))
	OnceMessage(string, func(*messages.Message)) (notification.Disposable, error)
	OnMessage(string, func(*messages.Message)) (notification.Disposable, error)
	OffMessage(string, func(*messages.Message))
	OffAll(string)
	OnError(func(error))
	OnClose(func(error))
	Close() error
}

func NewConnection(c connection.IWsConnection, timeout time.Duration, messageParser messages.IMessageParser, notifications notification.IWRNotificationEmitter) IConnection {
	if timeout < time.Second*15 {
		timeout = time.Second * 15
	} else if timeout > time.Second*60 {
		timeout = time.Second * 60
	}
	conn := &Connection{c, timeout, messageParser, notifications, nil}
	conn.ws.OnError(func(err error) {
		conn.ws.Close()
	})
	conn.ws.StartListening()
	conn.ws.OnMessage(func(stream []byte) {
		msg, err := conn.messageParser.Deserialize(stream)
		if err != nil {
			if notifications.HasEvent(msg.Id()) {
				notifications.Notify(msg.Id(), msg)
			} else if conn.messageCallback != nil {
				conn.messageCallback(msg)
			}
		}
	})
	return conn
}

func (c *Connection) Address() string {
	return c.ws.Address()
}

// AsyncRequest DO NOT RECOMMEND DUE TO LACK OF ERROR HINTS
func (c *Connection) AsyncRequest(message *messages.Message) (barrier *async.StatefulBarrier, err error) {
	barrier = async.NewStatefulBarrier()
	if err = c.Send(message); err != nil {
		return
	}
	timeoutEvent := timed.RunAsyncTimeout(func() {
		barrier.OpenWith(messages.NewErrorMessage(message.Id(), message.From(), message.To(), message.Uri(), "Handle timeout"))
	}, c.requestTimeout)
	c.notificationEmitter.Once(message.Id(), func(msg *messages.Message) {
		timed.Cancel(timeoutEvent)
		if msg == nil {
			barrier.OpenWith(messages.NewErrorMessage(message.Id(), message.From(), message.To(), message.Uri(), "invalid(nil) response for request "+message.Id()))
		} else {
			barrier.OpenWith(msg)
		}
	})
	return
}

// Request naive way to conduct async in Go to give better error hint
func (c *Connection) Request(message *messages.Message) (response *messages.Message, err error) {
	return c.RequestWithTimeout(message, c.requestTimeout)
}

func (c *Connection) RequestWithTimeout(message *messages.Message, timeout time.Duration) (response *messages.Message, err error) {
	waiter := make(chan bool)
	if err = c.Send(message); err != nil {
		return
	}
	timeoutEvent := timed.RunAsyncTimeout(func() {
		close(waiter)
	}, timeout)
	c.OnceMessage(message.Id(), func(msg *messages.Message) {
		timed.Cancel(timeoutEvent)
		if msg == nil {
			err = errors.New("invalid(nil) response for request " + message.Id())
		} else {
			response = msg
		}
		close(waiter)
	})
	<-waiter
	return
}

func (c *Connection) Send(message *messages.Message) error {
	if m, e := c.messageParser.Serialize(message); e == nil {
		return c.ws.Write(m)
	} else {
		return e
	}
}

func (c *Connection) OnIncomingMessage(cb func(*messages.Message)) {
	c.messageCallback = cb
}

func (c *Connection) OnMessage(id string, cb func(*messages.Message)) (notification.Disposable, error) {
	return c.notificationEmitter.On(id, cb)
}

func (c *Connection) OnceMessage(id string, cb func(*messages.Message)) (notification.Disposable, error) {
	return c.notificationEmitter.Once(id, cb)
}

func (c *Connection) OffMessage(id string, cb func(*messages.Message)) {
	c.notificationEmitter.Off(id, cb)
}

func (c *Connection) OffAll(id string) {
	c.notificationEmitter.OffAll(id)
}

func (c *Connection) Close() error {
	return c.ws.Close()
}

func (c *Connection) OnClose(cb func(error)) {
	c.ws.OnClose(cb)
}

func (c *Connection) OnError(cb func(error)) {
	c.ws.OnError(cb)
}
