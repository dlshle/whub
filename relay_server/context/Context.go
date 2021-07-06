package context

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

var Ctx *Context

func init() {
	Ctx = NewContext()
}

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
	startBarrier        *async.Barrier
	logger              *logger.SimpleLogger
}

type IContext interface {
	Server() roles.IDescribableRole
	TimedJobPool() *timed.JobPool
	NotificationEmitter() notification.IWRNotificationEmitter
	AsyncTaskPool() *async.AsyncPool
	MessageParser() messages.IMessageParser
	ServiceTaskPool() *async.AsyncPool
	Logger() *logger.SimpleLogger
}

func NewContext() *Context {
	asyncPool := async.NewAsyncPool(fmt.Sprintf("[ctx-async-pool]"), defaultAsyncPoolSize, runtime.NumCPU()*defaultAsyncPoolWorkerFactor)
	asyncPool.Verbose(false)
	servicePool := async.NewAsyncPool(fmt.Sprintf("[ctx-service-pool]"), defaultServicePoolSize, runtime.NumCPU()*defaultServicePoolWorkerFactor)
	servicePool.Verbose(false)
	jobPool := timed.NewJobPool("[ctx-timed-job-pool]", defaultTimedJobPoolSize, false)
	jobPool.Verbose(false)
	return &Context{
		messageParser:       messages.NewFBMessageParser(),
		asyncTaskPool:       asyncPool,
		serviceTaskPool:     servicePool,
		timedJobPool:        jobPool,
		notificationEmitter: notification.New(defaultMaxListenerCount),
		lock:                new(sync.RWMutex),
		startBarrier:        async.NewBarrier(),
		logger:              logger.New(os.Stdout, "[WServer]", true),
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
		c.startBarrier.Open()
	})
}

func (c *Context) Server() roles.IDescribableRole {
	c.startBarrier.Wait()
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

func (c *Context) Logger() *logger.SimpleLogger {
	return c.logger
}
