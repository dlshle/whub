package WRCommon

import (
	"github.com/dlshle/gommon/notification"
	"github.com/dlshle/gommon/timed"
	"sync/atomic"
)

var globalContext IWRContext

const (
	defaultTimedJobPoolSize = 4096
	defaultMaxListenerCount = 1024
)

type WRContext struct {
	identity IDescribableRole
	timedJobPool *timed.JobPool
	notificationEmitter notification.INotificationEmitter
	hasStarted atomic.Value
}

type IWRContext interface {
	Identity() IDescribableRole
	TimedJobPool() *timed.JobPool
	NotificationEmitter() notification.INotificationEmitter
	HasStarted() bool
}

func NewWRContext(role IDescribableRole, maxTimedJobs, maxNotificationListeners int) *WRContext {
	atomicBool := atomic.Value{}
	atomicBool.Store(false)
	return &WRContext{role, timed.NewJobPool("WRContext", maxTimedJobs, false), notification.New(maxNotificationListeners), atomicBool}
}

// TODO impls