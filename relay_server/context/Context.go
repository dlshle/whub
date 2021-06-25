package context

import (
	"fmt"
	"runtime"
	"sync"
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
	defaultAsyncPoolWorkerFactor   = 32
	defaultServicePoolWorkerFactor = 16
)

type Context struct {
	server              roles.ICommonServer
	asyncTaskPool       *async.AsyncPool
	serviceTaskPool     *async.AsyncPool
	timedJobPool        *timed.JobPool
	notificationEmitter notification.IWRNotificationEmitter
	messageParser       messages.IMessageParser
	lock                *sync.RWMutex
}

type IContext interface {
	Server() roles.IDescribableRole
	TimedJobPool() *timed.JobPool
	NotificationEmitter() notification.IWRNotificationEmitter
	AsyncTaskPool() *async.AsyncPool
	MessageParser() messages.IMessageParser
	ServiceTaskPool() *async.AsyncPool
}

func NewContext() *Context {
	asyncPool := async.NewAsyncPool(fmt.Sprintf("[ctx-async-pool]"), defaultAsyncPoolSize, runtime.NumCPU()*defaultAsyncPoolWorkerFactor)
	servicePool := async.NewAsyncPool(fmt.Sprintf("[ctx-service-pool]"), defaultServicePoolSize, runtime.NumCPU()*defaultServicePoolWorkerFactor)
	return &Context{
		messageParser:       messages.NewFBMessageParser(),
		asyncTaskPool:       asyncPool,
		serviceTaskPool:     servicePool,
		timedJobPool:        timed.NewJobPool("Context", defaultTimedJobPoolSize, false),
		notificationEmitter: notification.New(defaultMaxListenerCount),
		lock:                new(sync.RWMutex),
	}
}

func (c *Context) withWrite(cb func()) {
	c.lock.Lock()
	defer c.lock.Unlock()
	cb()
}

func (c *Context) Start(server roles.ICommonServer) {
	c.withWrite(func() {
		c.server = server
	})
}

func (c *Context) Server() roles.IDescribableRole {
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
