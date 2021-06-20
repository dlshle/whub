package relay_common

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"wsdk/common/async"
	"wsdk/common/timed"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/notification"
)

var globalContext IWRContext

const (
	defaultTimedJobPoolSize        = 4096
	defaultMaxListenerCount        = 1024
	defaultAsyncPoolSize           = 2048
	defaultServicePoolSize         = 1024
	defaultAsyncPoolWorkerFactor   = 16
	defaultServicePoolWorkerFactor = 8
)

type WRContext struct {
	identity            IDescribableRole
	asyncTaskPool       *async.AsyncPool
	serviceTaskPool     *async.AsyncPool
	timedJobPool        *timed.JobPool
	notificationEmitter notification.IWRNotificationEmitter
	hasStarted          atomic.Value
	messageParser       messages.IMessageParser
}

type IWRContext interface {
	Identity() IDescribableRole
	TimedJobPool() *timed.JobPool
	NotificationEmitter() notification.IWRNotificationEmitter
	AsyncTaskPool() *async.AsyncPool
	MessageParser() messages.IMessageParser
	ServiceTaskPool() *async.AsyncPool
	HasStarted() bool
	Start()
	Stop()
}

func NewWRContext(role IDescribableRole, conn *connection.WRConnection) *WRContext {
	atomicBool := atomic.Value{}
	atomicBool.Store(false)
	asyncPool := async.NewAsyncPool(fmt.Sprintf("[%s-ctx-async-pool]", role.Id()), 2048, runtime.NumCPU()*defaultAsyncPoolWorkerFactor)
	servicePool := async.NewAsyncPool(fmt.Sprintf("[%s-ctx-service-pool]", role.Id()), 1024, runtime.NumCPU()*defaultServicePoolWorkerFactor)
	return &WRContext{
		identity:            role,
		messageParser:       messages.NewFBMessageParser(),
		asyncTaskPool:       asyncPool,
		serviceTaskPool:     servicePool,
		timedJobPool:        timed.NewJobPool("WRContext", defaultTimedJobPoolSize, false),
		notificationEmitter: notification.New(defaultMaxListenerCount),
		hasStarted:          atomicBool,
	}
}

func (c *WRContext) Identity() IDescribableRole {
	return c.identity
}

func (c *WRContext) TimedJobPool() *timed.JobPool {
	return c.timedJobPool
}

func (c *WRContext) AsyncTaskPool() *async.AsyncPool {
	return c.asyncTaskPool
}

func (c *WRContext) ServiceTaskPool() *async.AsyncPool {
	return c.serviceTaskPool
}

func (c *WRContext) NotificationEmitter() notification.IWRNotificationEmitter {
	return c.notificationEmitter
}

func (c *WRContext) MessageParser() messages.IMessageParser {
	return c.messageParser
}

func (c *WRContext) HasStarted() bool {
	return c.hasStarted.Load().(bool)
}

func (c *WRContext) Start() {
	c.hasStarted.Store(true)
}

func (c *WRContext) Stop() {
	c.hasStarted.Store(false)
}
