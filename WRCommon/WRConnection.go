package WRCommon

import (
	"errors"
	"github.com/dlshle/gommon/async"
	"github.com/dlshle/gommon/timed"
	"github.com/dlshle/gommon/notification"
	"time"
	"wsdk/Common"
)

type WRConnection struct {
	c *Common.WsConnection
	requestTimeoutMills time.Duration
	messageHandler IMessageHandler
}

type IWRConnection interface {
	AsyncRequest(*Message) (*async.StatefulBarrier, error)
	Request(*Message) (*Message, error)
	Send(*Message) error
	// TODO how to observe on new message with current notification design?
	OnceMessage(string, func(*Message)) (notification.Disposable, error)
	OnMessage(string, func(*Message)) (notification.Disposable, error)
	OffMessage(string, func(*Message))
	OffAllMessage(string)
}

func NewWRConnection(c *Common.WsConnection, timeoutInMills time.Duration, messageHandler IMessageHandler) *WRConnection {
	conn := &WRConnection{c, timeoutInMills, messageHandler}
	conn.c.OnError(func(err error) {
		conn.c.Close()
	})
	conn.c.Start()
	conn.c.OnMessage(func(stream []byte) {
		msg, err := conn.messageHandler.Deserialize(stream)
		if err != nil {
			notification.Notify(msg.Id(), msg)
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
	notification.Once(message.Id(), func(msg interface{}) {
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
	notification.Once(message.Id(), func(msg interface{}) {
		timed.Cancel(timeoutEvent)
		if msg == nil {
			err = errors.New("invalid(nil) response for request " + message.Id())
		} else {
			response = msg.(*Message)
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

func (c *WRConnection) OnMessage(id string, cb func(*Message)) (notification.Disposable, error) {
	return notification.On(id, cb)
}