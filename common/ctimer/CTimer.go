package ctimer

import (
	"time"
)

const (
	StatusIdle = iota
	StatusWaiting
	StatusReset
	StatusCancelled
	StatusRunning
	StatusRepeatWaiting
	StatusRepeatRunning
)

type ICTimer interface {
	Start()
	Reset()
	Cancel()
	Repeat()
}

type CTimer struct {
	job           func()
	startTime     time.Time
	resetInterval time.Duration
	interval      time.Duration
	status        uint8
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
		go t.waitAndRun(t.interval)
	}
}

func (t *CTimer) Repeat() {
	if t.status == StatusIdle {
		go t.repeatWaitAndRun(t.interval)
	}
}

func (t *CTimer) repeatWaitAndRun(interval time.Duration) {
	for t.status != StatusCancelled {
		t.waitAndRun(interval)
	}
}

func (t *CTimer) waitAndRun(interval time.Duration) {
	if t.status == StatusCancelled {
		return
	}
	t.resetInterval = 0
	t.startTime = time.Now()
	t.status = StatusWaiting
	time.Sleep(interval)
	if t.status == StatusCancelled {
		t.status = StatusIdle
		return
	}
	if t.status == StatusReset && t.resetInterval > 0 {
		t.waitAndRun(t.resetInterval)
		return
	}
	t.status = StatusRunning
	t.job()
	t.status = StatusIdle
}

func (t *CTimer) Reset() {
	if t.status == StatusWaiting || t.status == StatusReset {
		t.status = StatusReset
		previousTime := t.startTime
		t.startTime = time.Now()
		t.resetInterval = t.resetInterval + t.startTime.Sub(previousTime)
		return
	} else {
		t.Start()
	}
}

func (t *CTimer) Cancel() {
	if t.status == StatusWaiting || t.status == StatusRepeatWaiting {
		t.status = StatusCancelled
	}
}
