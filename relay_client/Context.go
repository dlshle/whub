package relay_client

import (
	"fmt"
	"runtime"
	"wsdk/common/async"
	"wsdk/common/timed"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/notification"
	"wsdk/relay_common/roles"
)

const (
	defaultTimedJobPoolSize        = 4096
	defaultMaxListenerCount        = 1024
	defaultAsyncPoolSize           = 2048
	defaultServicePoolSize         = 1024
	defaultAsyncPoolWorkerFactor   = 16
	defaultServicePoolWorkerFactor = 8
)

type Context struct {
	identity            roles.IDescribableRole
	server              IServer
	asyncTaskPool       *async.AsyncPool
	serviceTaskPool     *async.AsyncPool
	timedJobPool        *timed.JobPool
	notificationEmitter notification.IWRNotificationEmitter
	messageParser       messages.IMessageParser
}

type IContext interface {
	Server() IServer
	Identity() roles.IDescribableRole
	TimedJobPool() *timed.JobPool
	NotificationEmitter() notification.IWRNotificationEmitter
	AsyncTaskPool() *async.AsyncPool
	MessageParser() messages.IMessageParser
	ServiceTaskPool() *async.AsyncPool
}

func NewContext() *Context {
	asyncPool := async.NewAsyncPool(fmt.Sprintf("[ctx-async-pool]"), 2048, runtime.NumCPU()*defaultAsyncPoolWorkerFactor)
	servicePool := async.NewAsyncPool(fmt.Sprintf("[ctx-service-pool]"), 1024, runtime.NumCPU()*defaultServicePoolWorkerFactor)
	return &Context{
		messageParser:       messages.NewFBMessageParser(),
		asyncTaskPool:       asyncPool,
		serviceTaskPool:     servicePool,
		timedJobPool:        timed.NewJobPool("Context", defaultTimedJobPoolSize, false),
		notificationEmitter: notification.New(defaultMaxListenerCount),
	}
}

func (c *Context) Start(identity roles.IDescribableRole, server IServer) {
	c.identity = identity
	c.server = server
}

func (c *Context) Identity() roles.IDescribableRole {
	return c.identity
}

func (c *Context) Server() IServer {
	return c.server
}

func (c *Context) TimedJobPool() *timed.JobPool {
	return c.timedJobPool
}

func (c *Context) AsyncTaskPool() *async.AsyncPool {
	return c.asyncTaskPool
}

func (c *Context) ServiceTaskPool() *async.AsyncPool {
	return c.serviceTaskPool
}

func (c *Context) NotificationEmitter() notification.IWRNotificationEmitter {
	return c.notificationEmitter
}

func (c *Context) MessageParser() messages.IMessageParser {
	return c.messageParser
}
