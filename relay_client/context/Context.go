package context

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sync"
	"wsdk/common/async"
	"wsdk/common/http"
	"wsdk/common/logger"
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
	defaultMaxActiveServiceConns   = 3
	defaultHTTPClientCount         = 5
	defaultHTTPClientMaxQueueSize  = 256
	defaultHTTPClientTimeout       = 60
)

type Context struct {
	lock                *sync.Mutex
	ctx                 context.Context
	cancelFunc          func()
	identity            roles.IDescribableRole
	server              roles.ICommonServer
	asyncTaskPool       async.IAsyncPool
	serviceTaskPool     async.IAsyncPool
	notificationEmitter notification.IWRNotificationEmitter
	messageParser       messages.IMessageParser
	logger              *logger.SimpleLogger
	httpClient          http.IClientPool
	startWaiter         *async.WaitLock
}

type IContext interface {
	Start(identity roles.IDescribableRole, server roles.ICommonServer)
	Identity() roles.IDescribableRole
	Server() roles.ICommonServer
	NotificationEmitter() notification.IWRNotificationEmitter
	AsyncTaskPool() async.IAsyncPool
	MessageParser() messages.IMessageParser
	ServiceTaskPool() async.IAsyncPool
	Logger() *logger.SimpleLogger
	MaxActiveServiceConnections() int
	HTTPClient() (pool http.IClientPool)
	Stop()
	Context() context.Context
}

func NewContext() IContext {
	ctx, cancelFunc := context.WithCancel(context.Background())
	return &Context{
		ctx:           ctx,
		cancelFunc:    cancelFunc,
		lock:          new(sync.Mutex),
		messageParser: messages.NewFBMessageParser(),
		logger:        logger.New(os.Stdout, "[WClient]", true),
		startWaiter:   async.NewWaitLock(),
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
	c.startWaiter.Open()
}

func (c *Context) Identity() roles.IDescribableRole {
	c.startWaiter.Wait()
	return c.identity
}

func (c *Context) Server() roles.ICommonServer {
	return c.server
}

func (c *Context) AsyncTaskPool() async.IAsyncPool {
	c.withLock(func() {
		if c.asyncTaskPool == nil {
			c.asyncTaskPool = async.NewAsyncPool(fmt.Sprintf("[ctx-async-pool]"), 2048, runtime.NumCPU()*defaultAsyncPoolWorkerFactor)
		}
	})
	return c.asyncTaskPool
}

func (c *Context) ServiceTaskPool() async.IAsyncPool {
	c.withLock(func() {
		if c.serviceTaskPool == nil {
			c.serviceTaskPool = async.NewAsyncPool(fmt.Sprintf("[ctx-service-pool]"), 1024, runtime.NumCPU()*defaultServicePoolWorkerFactor)
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

func (c *Context) Stop() {
	c.logger.Println("client has stopped")
	c.cancelFunc()
}

func (c *Context) Context() context.Context {
	return c.ctx
}

func (c *Context) MaxActiveServiceConnections() int {
	return defaultMaxActiveServiceConns
}

func (c *Context) HTTPClient() (pool http.IClientPool) {
	c.withLock(func() {
		if c.httpClient == nil {
			c.httpClient = http.NewPool("[HTTPClient]", defaultHTTPClientCount, defaultHTTPClientMaxQueueSize, defaultHTTPClientTimeout)
		}
		pool = c.httpClient
	})
	return
}
