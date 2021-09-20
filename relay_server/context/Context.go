package context

import (
	"context"
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
import . "wsdk/relay_server/config"

var Ctx *Context

func init() {
	Ctx = NewContext()
}

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
	config := Config.CommonConfig
	asyncPool := async.NewAsyncPool("[ctx-async-pool]", config.MaxAsyncPoolSize, runtime.NumCPU()*config.AsyncPoolWorkerFactor)
	asyncPool.Verbose(false)
	servicePool := async.NewAsyncPool("[ctx-service-pool]", config.MaxServiceAsyncPoolSize, runtime.NumCPU()*config.ServiceAsyncPoolWorkerFactor)
	servicePool.Verbose(false)
	ctx, cancel := context.WithCancel(context.Background())
	return &Context{
		messageParser:       messages.NewFBMessageParser(),
		asyncTaskPool:       asyncPool,
		serviceTaskPool:     servicePool,
		notificationEmitter: notification.New(config.MaxListenerCount),
		lock:                new(sync.Mutex),
		ctx:                 &ctx,
		cancelFunc:          cancel,
		startWaiter:         async.NewWaitLock(),
		logger:              logger.New(os.Stdout, "[WServer]", true),
		signKey:             ([]byte)(config.SignKey),
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
	return c.asyncTaskPool
}

func (c *Context) ServiceTaskPool() async.IAsyncPool {
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

func (c *Context) Stop() {
	c.cancelFunc()
}

func (c *Context) Context() *context.Context {
	return c.ctx
}

func (c *Context) SignKey() []byte {
	return c.signKey
}
