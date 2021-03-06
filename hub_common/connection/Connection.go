package connection

import (
	"errors"
	"fmt"
	"time"
	"whub/common/async"
	"whub/common/connection"
	"whub/common/ctimer"
	"whub/common/logger"
	"whub/hub_common/messages"
	"whub/hub_common/notification"
)

const WSConnectionPath = "/ws"

const DefaultTimeout = time.Second * 30
const DefaultAlivenessTimeout = time.Minute * 5

type Connection struct {
	conn                connection.IConnection
	address             string
	requestTimeout      time.Duration
	messageParser       messages.IMessageParser
	notificationEmitter notification.IWRNotificationEmitter
	messageCallback     func(messages.IMessage)
	logger              *logger.SimpleLogger
	ttlTimedJob         ctimer.ICTimer
}

type IConnection interface {
	Address() string
	ReadingLoop()
	Request(messages.IMessage) (messages.IMessage, error)
	RequestWithTimeout(messages.IMessage, time.Duration) (messages.IMessage, error)
	Send(messages.IMessage) error
	OnIncomingMessage(func(message messages.IMessage))
	OnceMessage(string, func(messages.IMessage)) (notification.Disposable, error)
	OnMessage(string, func(messages.IMessage)) (notification.Disposable, error)
	OffMessage(string, func(messages.IMessage))
	OffAll(string)
	OnError(func(error))
	OnClose(func(error))
	Close() error
	ConnectionType() uint8
	String() string
	IsLive() bool
}

func NewConnection(
	logger *logger.SimpleLogger,
	c connection.IConnection,
	timeout time.Duration,
	messageParser messages.IMessageParser,
	notifications notification.IWRNotificationEmitter,
) IConnection {
	if timeout < time.Second*15 {
		timeout = time.Second * 15
	} else if timeout > time.Second*60 {
		timeout = time.Second * 60
	}
	conn := &Connection{c, "", timeout, messageParser, notifications, nil, logger, nil}
	conn.ttlTimedJob = ctimer.New(DefaultAlivenessTimeout, conn.ttlJob)
	conn.conn.OnError(func(err error) {
		conn.conn.Close()
	})
	conn.conn.OnMessage(func(stream []byte) {
		conn.ttlTimedJob.Reset()
		msg, err := conn.messageParser.Deserialize(stream)
		if err == nil {
			if notifications.HasEvent(msg.Id()) {
				notifications.Notify(msg.Id(), msg)
			} else if conn.messageCallback != nil {
				conn.messageCallback(msg)
			}
		} else {
			logger.Println("unable to parse message ", stream)
		}
	})
	conn.ttlTimedJob.Start()
	return conn
}

func (conn *Connection) Init(
	logger *logger.SimpleLogger,
	c connection.IConnection,
	timeout time.Duration,
	messageParser messages.IMessageParser,
	notifications notification.IWRNotificationEmitter) {
	conn.conn = c
	conn.logger = logger
	conn.requestTimeout = timeout
	conn.messageParser = messageParser
	conn.notificationEmitter = notifications
	conn.ttlTimedJob = ctimer.New(DefaultAlivenessTimeout, conn.ttlJob)
	conn.conn.OnError(func(err error) {
		conn.conn.Close()
	})
	conn.conn.OnMessage(func(stream []byte) {
		conn.ttlTimedJob.Reset()
		msg, err := conn.messageParser.Deserialize(stream)
		if err == nil {
			if notifications.HasEvent(msg.Id()) {
				notifications.Notify(msg.Id(), msg)
			} else if conn.messageCallback != nil {
				conn.messageCallback(msg)
			}
		} else {
			logger.Println("unable to parse message ", stream)
		}
	})
	conn.address = c.Address()
	conn.ttlTimedJob.Start()
}

func (c *Connection) Address() string {
	if c.address == "" {
		c.address = c.conn.Address()
	}
	return c.address
}

func (c *Connection) ReadingLoop() {
	c.conn.ReadLoop()
}

// Request naive way to conduct async in Go to give better error hint
func (c *Connection) Request(message messages.IMessage) (response messages.IMessage, err error) {
	return c.RequestWithTimeout(message, c.requestTimeout)
}

func (c *Connection) RequestWithTimeout(message messages.IMessage, timeout time.Duration) (response messages.IMessage, err error) {
	barrier := async.NewWaitLock()
	if err = c.Send(message); err != nil {
		return
	}
	timeoutEvt := ctimer.New(timeout, func() {
		err = errors.New(fmt.Sprintf("request timeout for message %s", message.Id()))
		barrier.Open()
	})
	timeoutEvt.Start()
	c.OnceMessage(message.Id(), func(msg messages.IMessage) {
		timeoutEvt.Cancel()
		if msg == nil {
			err = errors.New("invalid(nil) response for request " + message.Id())
		} else {
			response = msg
		}
		barrier.Open()
	})
	barrier.Wait()
	return
}

func (c *Connection) Send(message messages.IMessage) (err error) {
	defer func() {
		if err != nil {
			c.logger.Println("write error: ", err)
		}
		message.Dispose()
	}()
	if m, e := c.messageParser.Serialize(message); e == nil {
		return c.conn.Write(m)
	} else {
		return e
	}
}

func (c *Connection) OnIncomingMessage(cb func(messages.IMessage)) {
	c.messageCallback = cb
}

func (c *Connection) OnMessage(id string, cb func(messages.IMessage)) (notification.Disposable, error) {
	return c.notificationEmitter.On(id, cb)
}

func (c *Connection) OnceMessage(id string, cb func(messages.IMessage)) (notification.Disposable, error) {
	return c.notificationEmitter.Once(id, cb)
}

func (c *Connection) OffMessage(id string, cb func(messages.IMessage)) {
	c.notificationEmitter.Off(id, cb)
}

func (c *Connection) OffAll(id string) {
	c.notificationEmitter.OffAll(id)
}

func (c *Connection) Close() error {
	return c.conn.Close()
}

func (c *Connection) OnClose(cb func(error)) {
	c.conn.OnClose(cb)
}

func (c *Connection) OnError(cb func(error)) {
	c.conn.OnError(cb)
}

func (c *Connection) ttlJob() {
	c.logger.Println("connection closed due to inactive timeout")
	c.Close()
}

func (c *Connection) ConnectionType() uint8 {
	return c.conn.ConnectionType()
}

func (c *Connection) String() string {
	return fmt.Sprintf("{\"type\":\"%s\",\"address\":\"%s\"}", c.TypeString(), c.Address())
}

func (c *Connection) TypeString() string {
	return connection.TypeString(c.ConnectionType())
}

func (c *Connection) IsLive() bool {
	return c.conn.IsLive()
}
