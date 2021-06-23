package relay_server

import (
	"fmt"
	"runtime"
	"sync"
	"wsdk/common/async"
	"wsdk/common/timed"
	"wsdk/relay_common"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/notification"
	"wsdk/relay_server/service"
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
	server         relay_common.IWRServer
	identity       relay_common.IDescribableRole
	clientManager  IClientManager
	serviceManager service.IServiceManager

	asyncTaskPool       *async.AsyncPool
	serviceTaskPool     *async.AsyncPool
	timedJobPool        *timed.JobPool
	notificationEmitter notification.IWRNotificationEmitter
	messageParser       messages.IMessageParser
	lock                *sync.RWMutex
}

type IContext interface {
	Identity() relay_common.IDescribableRole
	ClientManager() IClientManager
	ServiceManager() service.IServiceManager
	TimedJobPool() *timed.JobPool
	NotificationEmitter() notification.IWRNotificationEmitter
	AsyncTaskPool() *async.AsyncPool
	MessageParser() messages.IMessageParser
	ServiceTaskPool() *async.AsyncPool
}

func NewContext(identity relay_common.IDescribableRole) *Context {
	asyncPool := async.NewAsyncPool(fmt.Sprintf("[%s-ctx-async-pool]", identity.Id()), defaultAsyncPoolSize, runtime.NumCPU()*defaultAsyncPoolWorkerFactor)
	servicePool := async.NewAsyncPool(fmt.Sprintf("[%s-ctx-service-pool]", identity.Id()), defaultServicePoolSize, runtime.NumCPU()*defaultServicePoolWorkerFactor)
	return &Context{
		identity:            identity,
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

func (c *Context) Identity() relay_common.IDescribableRole {
	return c.identity
}

func (c *Context) TimedJobPool() *timed.JobPool {
	return c.timedJobPool
}

func (c *Context) ClientManager() IClientManager {
	c.withWrite(func() {
		if c.clientManager == nil {
			c.clientManager = NewClientManager(c)
		}
	})
	return c.clientManager
}

func (c *Context) ServiceManager() service.IServiceManager {
	c.withWrite(func() {
		if c.serviceManager == nil {
			c.serviceManager = service.NewServiceManager(c)
		}
	})
	return c.serviceManager
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
