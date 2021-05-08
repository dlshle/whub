package Connection

import (
	"errors"
	"github.com/dlshle/gommon/async"
	"github.com/dlshle/gommon/timed"
	"time"
	"wsdk/WRCommon/Message"
	"wsdk/WRCommon/Notification"
	Common2 "wsdk/base/Common"
)

type WRConnection struct {
	c                   *Common2.WsConnection
	requestTimeoutMills time.Duration
	messageHandler      Message.IMessageHandler
	notificationEmitter Notification.IWRNotificationEmitter
}

type IWRConnection interface {
	AsyncRequest(*Message.Message) (*async.StatefulBarrier, error)
	Request(*Message.Message) (*Message.Message, error)
	Send(*Message.Message) error
	OnceMessage(string, func(*Message.Message)) (Notification.Disposable, error)
	OnMessage(string, func(*Message.Message)) (Notification.Disposable, error)
	OffMessage(string, func(*Message.Message))
	OffAll(string)
	Close() error
}

func NewWRConnection(c *Common2.WsConnection, timeoutInMills time.Duration, messageHandler Message.IMessageHandler, notifications Notification.IWRNotificationEmitter) *WRConnection {
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
func (c *WRConnection) AsyncRequest(message *Message.Message) (barrier *async.StatefulBarrier, err error) {
	barrier = async.NewStatefulBarrier()
	if err = c.Send(message); err != nil {
		return
	}
	timeoutEvent := timed.RunAsyncTimeout(func() {
		barrier.OpenWith(Message.NewErrorMessage(message.Id(), message.From(), message.To(), "Request timeout"))
	}, c.requestTimeoutMills)
	c.notificationEmitter.Once(message.Id(), func(msg *Message.Message) {
		timed.Cancel(timeoutEvent)
		if msg == nil {
			barrier.OpenWith(Message.NewErrorMessage(message.Id(), message.From(), message.To(), "invalid(nil) response for request " + message.Id()))
		} else {
			barrier.OpenWith(msg)
		}
	})
	return
}

// Request naive way to conduct async in Go to give better error hint
func (c *WRConnection) Request(message *Message.Message) (response *Message.Message, err error) {
	waiter := make(chan bool)
	if err = c.Send(message); err != nil {
		return
	}
	timeoutEvent := timed.RunAsyncTimeout(func() {
		close(waiter)
	}, c.requestTimeoutMills)
	c.OnceMessage(message.Id(), func(msg *Message.Message) {
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

func (c *WRConnection) Send(message *Message.Message) error {
	if m, e := c.messageHandler.Serialize(message); e == nil {
		return c.c.Write(m)
	} else {
		return e
	}
}

func (c *WRConnection) OnMessage(id string, cb func(*Message.Message)) (Notification.Disposable, error) {
	return c.notificationEmitter.On(id, cb)
}

func (c *WRConnection) OnceMessage(id string, cb func(*Message.Message)) (Notification.Disposable, error) {
	return c.notificationEmitter.Once(id, cb)
}

func (c *WRConnection) OffMessage(id string, cb func(*Message.Message)) {
	c.notificationEmitter.Off(id, cb)
}

func (c *WRConnection) OffAll(id string) {
	c.notificationEmitter.OffAll(id)
}

func (c *WRConnection) Close() error {
	return c.c.Close()
}
