package WRCommon

import (
	"github.com/dlshle/gommon/notification"
	"github.com/dlshle/gommon/timed"
	"sync/atomic"
)

var globalContext IGlobalContext

const (
	defaultTimedJobPoolSize = 4096
	defaultMaxListenerCount = 1024
)

type GlobalContext struct {
	identity IDescribableRole
	timedJobPool *timed.JobPool
	notificationEmitter notification.INotificationEmitter
	hasStarted atomic.Value
}

type IGlobalContext interface {
	CurrentIdentity() IDescribableRole
	TimedJobPool() *timed.JobPool
	NotificationEmitter() notification.INotificationEmitter
	HasStarted() bool
}

func InitializeGlobalContext(identity IDescribableRole, timedJobPoolSize int, maxListenerCount int) {
	if globalContext == nil {
		gctx := &GlobalContext{identity, timed.NewJobPool("WRSDK", timedJobPoolSize, false), notification.New(maxListenerCount), atomic.Value{}}
		// TODO globalContext = gctx
	}
}

func validateAndGet(target interface{}, getter func() interface{}) interface{} {
	if target == nil {
		return nil
	}
	return getter()
}

func CurrentIdentity() IDescribableRole {
	return validateAndGet(globalContext, func() interface{} {
		return globalContext.CurrentIdentity()
	}).(IDescribableRole)
}
