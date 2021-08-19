package context

import (
	"context"
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
	defaultMaxListenerCount        = 1024
	defaultAsyncPoolSize           = 2048
	defaultServicePoolSize         = 1024
	defaultAsyncPoolWorkerFactor   = 32
	defaultServicePoolWorkerFactor = 16
	defaultMaxConcurrentConnection = 2048
	defaultMaxMiddlewareCount      = 64
	defaultSignKey                 = "d1s7218U7!d-r5b"
)

type Context struct {
	lock                *sync.Mutex
	ctx                 *context.Context
	cancelFunc          func()
	server              roles.ICommonServer
	asyncTaskPool       async.IAsyncPool
	serviceTaskPool     async.IAsyncPool
	notificationEmitter notification.IWRNotificationEmitter
	messageParser       messages.IMessageParser
	startWaiter         *async.WaitLock
	logger              *logger.SimpleLogger
	signKey             []byte
}

type IContext interface {
	Server() roles.IDescribableRole
	TimedJobPool() *timed.JobPool
	NotificationEmitter() notification.IWRNotificationEmitter
	AsyncTaskPool() async.IAsyncPool
	MessageParser() messages.IMessageParser
	ServiceTaskPool() async.IAsyncPool
	Logger() *logger.SimpleLogger
	Stop()
}

func NewContext() *Context {
	// TODO use config values from container
	asyncPool := async.NewAsyncPool("[ctx-async-pool]", defaultAsyncPoolSize, runtime.NumCPU()*defaultAsyncPoolWorkerFactor)
	asyncPool.Verbose(false)
	servicePool := async.NewAsyncPool("[ctx-service-pool]", defaultServicePoolSize, runtime.NumCPU()*defaultServicePoolWorkerFactor)
	servicePool.Verbose(false)
	ctx, cancel := context.WithCancel(context.Background())
	return &Context{
		messageParser:       messages.NewFBMessageParser(),
		asyncTaskPool:       asyncPool,
		serviceTaskPool:     servicePool,
		notificationEmitter: notification.New(defaultMaxListenerCount),
		lock:                new(sync.Mutex),
		ctx:                 &ctx,
		cancelFunc:          cancel,
		startWaiter:         async.NewWaitLock(),
		logger:              logger.New(os.Stdout, "[WServer]", true),
		signKey:             ([]byte)(defaultSignKey),
	}
}

func (c *Context) withLock(cb func()) {
	c.lock.Lock()
	defer c.lock.Unlock()
	cb()
}

func (c *Context) Start(server roles.ICommonServer) {
	c.withLock(func() {
		c.server = server
		c.startWaiter.Open()
	})
}

func (c *Context) Server() roles.IDescribableRole {
	c.startWaiter.Wait()
	return c.server
}

func (c *Context) AsyncTaskPool() async.IAsyncPool {
	c.withLock(func() {
		if c.asyncTaskPool == nil {
			workerSize := runtime.NumCPU() * defaultAsyncPoolWorkerFactor
			c.logger.Printf("async pool initialized with maxPoolSize %d and workerSize %d", defaultAsyncPoolSize, workerSize)
			c.asyncTaskPool = async.NewAsyncPool(fmt.Sprintf("[ctx-async-pool]"), defaultAsyncPoolSize, workerSize)
		}
	})
	return c.asyncTaskPool
}

func (c *Context) ServiceTaskPool() async.IAsyncPool {
	c.withLock(func() {
		if c.serviceTaskPool == nil {
			workerSize := runtime.NumCPU() * defaultServicePoolWorkerFactor
			c.logger.Printf("service async pool initialized with maxPoolSize %d and workerSize %d", defaultServicePoolSize, workerSize)
			c.serviceTaskPool = async.NewAsyncPool(fmt.Sprintf("[ctx-service-pool]"), defaultServicePoolSize, workerSize)
		}
	})
	return c.serviceTaskPool
}

func (c *Context) NotificationEmitter() notification.IWRNotificationEmitter {
	c.withLock(func() {
		if c.notificationEmitter == nil {
			c.logger.Printf("notificationEmitter has been initialized with maxListenerCount %d", defaultMaxListenerCount)
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

func (c *Context) Stop() {
	c.cancelFunc()
}

func (c *Context) Context() *context.Context {
	return c.ctx
}

func (c *Context) SignKey() []byte {
	return c.signKey
}
