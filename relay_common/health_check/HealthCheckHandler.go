package health_check

import (
	"time"
	"wsdk/common/ctimer"
)

const (
	DefaultHealthCheckInterval = time.Minute * 5
	MinimalHealthCheckInterval = time.Second * 5
	MaximumHealthCheckInterval = time.Minute * 15
)

type HealthCheckHandler struct {
	healthCheckTimer              ctimer.ICTimer
	onRetry                       bool
	onHealthCheckFailedCallback   func()
	onHealthCheckRestoredCallback func()
	healthCheckExecutor           func() error
	healthCheckInterval           time.Duration
}

func NewHealthCheckHandler(interval time.Duration, executor func() error, onFailed func(), onRestored func()) *HealthCheckHandler {
	return &HealthCheckHandler{
		onRetry:                       false,
		onHealthCheckFailedCallback:   onFailed,
		onHealthCheckRestoredCallback: onRestored,
		healthCheckExecutor:           executor,
		healthCheckInterval:           interval,
	}
}

func (h *HealthCheckHandler) OnHealthCheckFails(cb func()) {
	h.onHealthCheckFailedCallback = cb
}

func (h *HealthCheckHandler) OnHealthCheckRestored(cb func()) {
	h.onHealthCheckRestoredCallback = cb
}

func (h *HealthCheckHandler) StartHealthCheck() {
	if h.healthCheckTimer == nil {
		h.healthCheckTimer = ctimer.New(h.healthCheckInterval, h.doHealthCheck)
	}
	h.healthCheckTimer.Start()
}

func (h *HealthCheckHandler) doHealthCheck() {
	err := h.healthCheckExecutor()
	if err != nil {
		h.onRetry = true
		if h.onHealthCheckFailedCallback != nil {
			h.onHealthCheckFailedCallback()
		}
	} else if h.onRetry {
		// if err == nil && onRetry
		if h.onHealthCheckRestoredCallback != nil {
			h.onHealthCheckRestoredCallback()
		}
		h.onRetry = false
	}
}

func (h *HealthCheckHandler) StopHealthCheck() {
	if h.healthCheckTimer != nil {
		h.healthCheckTimer.Cancel()
	}
}

func (h *HealthCheckHandler) RestartHealthCheck() {
	h.StopHealthCheck()
	h.StartHealthCheck()
}

func (h *HealthCheckHandler) UpdateHealthCheckInterval(interval time.Duration) {
	if interval < MinimalHealthCheckInterval {
		interval = MinimalHealthCheckInterval
	} else if interval > MaximumHealthCheckInterval {
		interval = MaximumHealthCheckInterval
	}
	h.healthCheckInterval = interval
	h.RestartHealthCheck()
}

func (h *HealthCheckHandler) IsJobScheduled() bool {
	return h.healthCheckTimer != nil
}

func (h *HealthCheckHandler) SetHealthCheckExecutor(executor func() error) {
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
