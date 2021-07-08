package ctimer

import "time"

const (
	StatusIdle = iota
	StatusWaiting
	StatusReset
	StatusCancelled
	StatusRunning
)

type ICTimer interface {
	Start()
	Reset()
	Cancel()
}

type CTimer struct {
	job       func()
	startTime time.Time
	resetTime time.Time
	interval  time.Duration
	status    uint8
}

func New(interval time.Duration, job func()) ICTimer {
	return &CTimer{
		job:      job,
		interval: interval,
		status:   StatusIdle,
	}
}

func (t *CTimer) Start() {
	if t.status == StatusIdle {
		go t.WaitAndRun(t.interval)
	}
}

func (t *CTimer) WaitAndRun(interval time.Duration) {
	t.startTime = time.Now()
	t.status = StatusWaiting
	time.Sleep(interval)
	if t.status == StatusCancelled {
		t.status = StatusIdle
		return
	}
	if t.status == StatusReset && !t.resetTime.IsZero() {
		t.WaitAndRun(t.resetTime.Sub(t.startTime))
		return
	}
	t.status = StatusRunning
	t.job()
	t.status = StatusIdle
}

func (t *CTimer) Reset() {
	if t.status == StatusWaiting {
		t.status = StatusReset
		t.resetTime = time.Now()
		return
	} else {
		t.Start()
	}
}

func (t *CTimer) Cancel() {
	if t.status == StatusWaiting {
		t.status = StatusCancelled
	}
}
