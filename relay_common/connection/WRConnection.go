package connection

import (
	"errors"
	"github.com/dlshle/gommon/async"
	"github.com/dlshle/gommon/timed"
	"time"
	Common "wsdk/base/common"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/notification"
)

const DefaultTimeout = time.Second * 30

type WRConnection struct {
	*Common.WsConnection
	requestTimeout      time.Duration
	messageHandler      messages.IMessageHandler
	notificationEmitter notification.IWRNotificationEmitter
}

type IWRConnection interface {
	AsyncRequest(*messages.Message) (*async.StatefulBarrier, error)
	Request(*messages.Message) (*messages.Message, error)
	RequestWithTimeout(*messages.Message, time.Duration) (*messages.Message, error)
	Send(*messages.Message) error
	OnceMessage(string, func(*messages.Message)) (notification.Disposable, error)
	OnMessage(string, func(*messages.Message)) (notification.Disposable, error)
	OffMessage(string, func(*messages.Message))
	OffAll(string)
	Close() error
}

func NewWRConnection(c *Common.WsConnection, timeout time.Duration, messageHandler messages.IMessageHandler, notifications notification.IWRNotificationEmitter) *WRConnection {
	if timeout < time.Second * 15 {
		timeout = time.Second * 15
	} else if timeout > time.Second * 60 {
		timeout = time.Second * 60
	}
	conn := &WRConnection{c, timeout, messageHandler, notifications}
	conn.OnError(func(err error) {
		conn.WsConnection.Close()
	})
	conn.WsConnection.StartListening()
	conn.WsConnection.OnMessage(func(stream []byte) {
		msg, err := conn.messageHandler.Deserialize(stream)
		if err != nil {
			notifications.Notify(msg.Id(), msg)
		}
	})
	return conn
}

// AsyncRequest DO NOT RECOMMEND DUE TO LACK OF ERROR HINTS
func (c *WRConnection) AsyncRequest(message *messages.Message) (barrier *async.StatefulBarrier, err error) {
	barrier = async.NewStatefulBarrier()
	if err = c.Send(message); err != nil {
		return
	}
	timeoutEvent := timed.RunAsyncTimeout(func() {
		barrier.OpenWith(messages.NewErrorMessage(message.Id(), message.From(), message.To(), "Request timeout"))
	}, c.requestTimeout)
	c.notificationEmitter.Once(message.Id(), func(msg *messages.Message) {
		timed.Cancel(timeoutEvent)
		if msg == nil {
			barrier.OpenWith(messages.NewErrorMessage(message.Id(), message.From(), message.To(), "invalid(nil) response for request " + message.Id()))
		} else {
			barrier.OpenWith(msg)
		}
	})
	return
}

// Request naive way to conduct async in Go to give better error hint
func (c *WRConnection) Request(message *messages.Message) (response *messages.Message, err error) {
	return c.RequestWithTimeout(message, c.requestTimeout)
}

func (c *WRConnection) RequestWithTimeout(message *messages.Message, timeout time.Duration) (response *messages.Message, err error) {
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
	<- waiter
	return
}

func (c *WRConnection) Send(message *messages.Message) error {
	if m, e := c.messageHandler.Serialize(message); e == nil {
		return c.Write(m)
	} else {
		return e
	}
}

func (c *WRConnection) OnMessage(id string, cb func(*messages.Message)) (notification.Disposable, error) {
	return c.notificationEmitter.On(id, cb)
}

func (c *WRConnection) OnceMessage(id string, cb func(*messages.Message)) (notification.Disposable, error) {
	return c.notificationEmitter.Once(id, cb)
}

func (c *WRConnection) OffMessage(id string, cb func(*messages.Message)) {
	c.notificationEmitter.Off(id, cb)
}

func (c *WRConnection) OffAll(id string) {
	c.notificationEmitter.OffAll(id)
}

func (c *WRConnection) Close() error {
	return c.WsConnection.Close()
}
