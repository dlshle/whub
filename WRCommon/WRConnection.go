package WRCommon

import (
	"errors"
	"github.com/dlshle/gommon/async"
	"github.com/dlshle/gommon/timed"
	"time"
	"wsdk/Common"
)

type WRConnection struct {
	c *Common.WsConnection
	requestTimeoutMills time.Duration
	messageHandler IMessageHandler
	notificationEmitter IMessageNotificationEmitter
}

type IWRConnection interface {
	AsyncRequest(*Message) (*async.StatefulBarrier, error)
	Request(*Message) (*Message, error)
	Send(*Message) error
	OnceMessage(string, func(*Message)) (Disposable, error)
	OnMessage(string, func(*Message)) (Disposable, error)
	OffMessage(string, func(*Message))
	OffAll(string)
	Close() error
}

func NewWRConnection(c *Common.WsConnection, timeoutInMills time.Duration, messageHandler IMessageHandler, notifications IMessageNotificationEmitter) *WRConnection {
	conn := &WRConnection{c, timeoutInMills, messageHandler, notifications}
	conn.c.OnError(func(err error) {
		conn.c.Close()
	})
	conn.c.StartListening()
	conn.c.OnMessage(func(stream []byte) {
		msg, err := conn.messageHandler.Deserialize(stream)
		if err != nil {
			notifications.Notify(msg.Id(), msg)
		}
	})
	return conn
}

// AsyncRequest DO NOT RECOMMEND DUE TO LACK OF ERROR HINTS
func (c *WRConnection) AsyncRequest(message *Message) (barrier *async.StatefulBarrier, err error) {
	barrier = async.NewStatefulBarrier()
	if err = c.Send(message); err != nil {
		return
	}
	timeoutEvent := timed.RunAsyncTimeout(func() {
		barrier.OpenWith(NewErrorMessage(message.Id(), message.From(), message.To(), "Request timeout"))
	}, c.requestTimeoutMills)
	c.notificationEmitter.Once(message.Id(), func(msg *Message) {
		timed.Cancel(timeoutEvent)
		if msg == nil {
			barrier.OpenWith(NewErrorMessage(message.Id(), message.From(), message.To(), "invalid(nil) response for request " + message.Id()))
		} else {
			barrier.OpenWith(msg)
		}
	})
	return
}

// Request naive way to conduct async in Go to give better error hint
func (c *WRConnection) Request(message *Message) (response *Message, err error) {
	waiter := make(chan bool)
	if err = c.Send(message); err != nil {
		return
	}
	timeoutEvent := timed.RunAsyncTimeout(func() {
		close(waiter)
	}, c.requestTimeoutMills)
	c.OnceMessage(message.Id(), func(msg *Message) {
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

func (c *WRConnection) Send(message *Message) error {
	if m, e := c.messageHandler.Serialize(message); e == nil {
		return c.c.Write(m)
	} else {
		return e
	}
}

func (c *WRConnection) OnMessage(id string, cb func(*Message)) (Disposable, error) {
	return c.notificationEmitter.On(id, cb)
}

func (c *WRConnection) OnceMessage(id string, cb func(*Message)) (Disposable, error) {
	return c.notificationEmitter.Once(id, cb)
}

func (c *WRConnection) OffMessage(id string, cb func(*Message)) {
	c.notificationEmitter.Off(id, cb)
}

func (c *WRConnection) OffAll(id string) {
	c.notificationEmitter.OffAll(id)
}

func (c *WRConnection) Close() error {
	return c.c.Close()
}
