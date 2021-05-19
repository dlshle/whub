package service

import (
	"github.com/dlshle/gommon/timed"
	"time"
)

const (
	MinimalHealthCheckInterval = time.Second * 5
	MaximumHealthCheckInterval = time.Minute * 15
)

type ServiceHealthCheckHandler struct {
	timedJobPool *timed.JobPool
	healthCheckJobId int64
	onHealthCheckFailedCallback func()
	onHealthCheckRestoredCallback func()
	healthCheckExecutor func() error
	healthCheckInterval time.Duration
}

func NewServiceHealthCheckHandler(pool *timed.JobPool, interval time.Duration, executor func() error, onFailed func(), onRestored func()) *ServiceHealthCheckHandler {
	return &ServiceHealthCheckHandler{
		timedJobPool: pool,
		healthCheckJobId: -1,
		onHealthCheckFailedCallback: onFailed,
		onHealthCheckRestoredCallback: onRestored,
		healthCheckExecutor: executor,
		healthCheckInterval: interval,
	}
}

func (h *ServiceHealthCheckHandler) OnHealthCheckFails(cb func()) {
	h.onHealthCheckFailedCallback = cb
}

func (h *ServiceHealthCheckHandler) OnHealthCheckRestored(cb func()) {
	h.onHealthCheckRestoredCallback = cb
}

func (h *ServiceHealthCheckHandler) StartHealthCheck() {
	if h.healthCheckJobId != -1 {
		return
	}
	onRetry := false
	h.healthCheckJobId = h.timedJobPool.ScheduleAsyncIntervalJob(func() {
		err := h.healthCheckExecutor()
		if err != nil {
			onRetry = true
			if h.onHealthCheckFailedCallback != nil{
				h.onHealthCheckFailedCallback()
			}
		} else if onRetry {
			// if err == nil && onRetry
			if h.onHealthCheckRestoredCallback != nil {
				h.onHealthCheckRestoredCallback()
			}
			onRetry = false
		}
	}, h.healthCheckInterval)
}

func (h *ServiceHealthCheckHandler) StopHealthCheck() {
	if h.healthCheckJobId != -1 {
		h.timedJobPool.CancelJob(h.healthCheckJobId)
		h.healthCheckJobId = 0
	}
}

func (h *ServiceHealthCheckHandler) RestartHealthCheck() {
	if h.healthCheckJobId == -1 {
		h.StartHealthCheck()
		return
	}
	h.StopHealthCheck()
	h.StartHealthCheck()
}

func (h *ServiceHealthCheckHandler) UpdateHealthCheckInterval(interval time.Duration) {
	if interval < MinimalHealthCheckInterval {
		interval = MinimalHealthCheckInterval
	} else if interval > MaximumHealthCheckInterval {
		interval = MaximumHealthCheckInterval
	}
	h.healthCheckInterval = interval
	h.RestartHealthCheck()
}

func (h *ServiceHealthCheckHandler) IsJobScheduled() bool {
	return h.healthCheckJobId > -1
}

func (h *ServiceHealthCheckHandler) SetHealthCheckExecutor(executor func() error) {
	if executor != nil {
		if h.IsJobScheduled() {
			h.StopHealthCheck()
		}
		h.healthCheckExecutor = executor
		if h.IsJobScheduled() {
			h.StartHealthCheck()
		}
	}
}