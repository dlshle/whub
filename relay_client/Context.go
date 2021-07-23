package relay_client

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"wsdk/common/async"
	"wsdk/common/logger"
	"wsdk/common/timed"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/notification"
	"wsdk/relay_common/roles"
)

var Ctx IContext

func init() {
	Ctx = NewContext()
}

const (
	defaultTimedJobPoolSize        = 4096
	defaultMaxListenerCount        = 1024
	defaultAsyncPoolSize           = 2048
	defaultServicePoolSize         = 1024
	defaultAsyncPoolWorkerFactor   = 16
	defaultServicePoolWorkerFactor = 8
)

type Context struct {
	lock                *sync.Mutex
	identity            roles.IDescribableRole
	server              roles.ICommonServer
	asyncTaskPool       *async.AsyncPool
	serviceTaskPool     *async.AsyncPool
	timedJobPool        *timed.JobPool
	notificationEmitter notification.IWRNotificationEmitter
	messageParser       messages.IMessageParser
	logger              *logger.SimpleLogger
	barrier             *async.Barrier
}

type IContext interface {
	Start(identity roles.IDescribableRole, server roles.ICommonServer)
	Server() roles.ICommonServer
	Identity() roles.IDescribableRole
	TimedJobPool() *timed.JobPool
	NotificationEmitter() notification.IWRNotificationEmitter
	AsyncTaskPool() *async.AsyncPool
	MessageParser() messages.IMessageParser
	ServiceTaskPool() *async.AsyncPool
	Logger() *logger.SimpleLogger
}

func NewContext() IContext {
	return &Context{
		lock:          new(sync.Mutex),
		messageParser: messages.NewFBMessageParser(),
		logger:        logger.New(os.Stdout, "[WClient]", true),
		barrier:       async.NewBarrier(),
	}
}

func (c *Context) withLock(cb func()) {
	c.lock.Lock()
	defer c.lock.Unlock()
	cb()
}

func (c *Context) Start(identity roles.IDescribableRole, server roles.ICommonServer) {
	c.identity = identity
	c.server = server
	c.barrier.Open()
}

func (c *Context) Identity() roles.IDescribableRole {
	c.barrier.Wait()
	return c.identity
}

func (c *Context) Server() roles.ICommonServer {
	return c.server
}

func (c *Context) TimedJobPool() *timed.JobPool {
	c.withLock(func() {
		if c.timedJobPool == nil {
			c.timedJobPool = timed.NewJobPool("Context", defaultTimedJobPoolSize, false)
		}
	})
	return c.timedJobPool
}

func (c *Context) AsyncTaskPool() *async.AsyncPool {
	c.withLock(func() {
		if c.asyncTaskPool == nil {
			c.asyncTaskPool = async.NewAsyncPool(fmt.Sprintf("[ctx-async-pool]"), 2048, runtime.NumCPU()*defaultAsyncPoolWorkerFactor)
		}
	})
	return c.asyncTaskPool
}

func (c *Context) ServiceTaskPool() *async.AsyncPool {
	c.withLock(func() {
		if c.serviceTaskPool == nil {
			c.serviceTaskPool = async.NewAsyncPool(fmt.Sprintf("[ctx-service_manager-pool]"), 1024, runtime.NumCPU()*defaultServicePoolWorkerFactor)
		}
	})
	return c.serviceTaskPool
}

func (c *Context) NotificationEmitter() notification.IWRNotificationEmitter {
	c.withLock(func() {
		if c.notificationEmitter == nil {
			c.notificationEmitter = notification.New(defaultMaxListenerCount)
		}
	})
	return c.notificationEmitter
}

func (c *Context) MessageParser() messages.IMessageParser {
	return c.messageParser
}

func (c *Context) Logger() *logger.SimpleLogger {
	return c.logger
}
