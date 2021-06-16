package relay_common

import (
	"sync/atomic"
	"wsdk/common/timed"
	"wsdk/relay_common/notification"
)

var globalContext IWRContext

const (
	defaultTimedJobPoolSize = 4096
	defaultMaxListenerCount = 1024
)

type WRContext struct {
	identity            IDescribableRole
	timedJobPool        *timed.JobPool
	notificationEmitter notification.IWRNotificationEmitter
	hasStarted          atomic.Value
}

type IWRContext interface {
	Identity() IDescribableRole
	TimedJobPool() *timed.JobPool
	NotificationEmitter() notification.IWRNotificationEmitter
	HasStarted() bool
	Start()
	Stop()
}

func NewWRContext(role IDescribableRole, maxTimedJobs, maxNotificationListeners int) *WRContext {
	atomicBool := atomic.Value{}
	atomicBool.Store(false)
	return &WRContext{role, timed.NewJobPool("WRContext", maxTimedJobs, false), notification.New(maxNotificationListeners), atomicBool}
}

func (c *WRContext) Identity() IDescribableRole {
	return c.identity
}

func (c *WRContext) TimedJobPool() *timed.JobPool {
	return c.timedJobPool
}

func (c *WRContext) NotificationEmitter() notification.IWRNotificationEmitter {
	return c.notificationEmitter
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
