package messages

import (
	"errors"
	"github.com/dlshle/gommon/async"
	"sync"
	"wsdk/relay_common/connection"
)

func init() {
	initServiceMessage()
}

const (
	ServiceMessageStatusQueued     = 0
	ServiceMessageStatusProcessing = 1
	ServiceMessageStatusDead       = 2 // when health check failed
	ServiceMessageStatusFinished   = 3
	ServiceMessageStatusCancelled  = 4
)

var unProcessableServiceMessageMap map[int]bool
var statusCodeStringMap map[int]string

func initServiceMessage() {
	statusCodeStringMap = make(map[int]string)
	statusCodeStringMap[ServiceMessageStatusQueued] = "queued"
	statusCodeStringMap[ServiceMessageStatusProcessing] = "processing"
	statusCodeStringMap[ServiceMessageStatusDead] = "dead"
	statusCodeStringMap[ServiceMessageStatusFinished] = "finished"
	statusCodeStringMap[ServiceMessageStatusCancelled] = "cancelled"

	unProcessableServiceMessageMap = make(map[int]bool)
	unProcessableServiceMessageMap[ServiceMessageStatusDead] = true
	unProcessableServiceMessageMap[ServiceMessageStatusCancelled] = true
}

type ServiceMessage struct {
	barrier *async.StatefulBarrier
	status  int
	lock    *sync.RWMutex
	*Message
	onStatusChangeCallback func(int)
}

func NewServiceMessage(m *Message) *ServiceMessage {
	return &ServiceMessage{async.NewStatefulBarrier(), ServiceMessageStatusQueued, new(sync.RWMutex), m, nil}
}

type IServiceMessage interface {
	Id() string
	Status() int
	Kill() error
	Cancel() error
	IsDead() bool
	IsCancelled() bool
	IsFinished() bool
	OnStatusChange(func(int))
	resolve(*Message) error
	Wait() error // wait for the state to transit to final (dead/finished/cancelled)
	Response() *Message
}

func (t *ServiceMessage) withWrite(cb func()) {
	t.lock.Lock()
	defer t.lock.Unlock()
	cb()
}

func (t *ServiceMessage) setStatus(status int) {
	if t.Status() == ServiceMessageStatusFinished {
		// can not set status of a finished service messages
		return
	}
	t.withWrite(func() {
		t.status = status
	})
	if t.onStatusChangeCallback != nil {
		t.onStatusChangeCallback(status)
	}
}


func (t *ServiceMessage) Status() int {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.status
}

func (t *ServiceMessage) Kill() error {
	if t.Status() > 1 {
		return errors.New("unable to kill a " + statusCodeStringMap[t.Status()] + " ServiceMessage")
	}
	t.withWrite(func() {
		t.status = ServiceMessageStatusDead
		t.barrier.OpenWith(nil)
	})
	return nil
}

func (t *ServiceMessage) Cancel() error {
	if t.Status() > 1 {
		return errors.New("unable to cancel a " + statusCodeStringMap[t.Status()] + " ServiceMessage")
	}
	t.withWrite(func() {
		t.status = ServiceMessageStatusCancelled
		t.barrier.OpenWith(nil)
	})
	return nil
}

func (t *ServiceMessage) resolve(m *Message) error {
	if t.Status() != ServiceMessageStatusProcessing {
		return errors.New("can not resolve a non-processing ServiceMessage")
	}
	t.withWrite(func() {
		t.status = ServiceMessageStatusFinished
		t.barrier.OpenWith(m)
	})
	return nil
}

func (t *ServiceMessage) IsDead() bool {
	return t.Status() == ServiceMessageStatusDead
}

func (t *ServiceMessage) IsCancelled() bool {
	return t.Status() == ServiceMessageStatusCancelled
}

func (t *ServiceMessage) IsFinished() bool {
	return t.Status() == ServiceMessageStatusFinished
}

func (t *ServiceMessage) OnStatusChange(cb func(int)) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.onStatusChangeCallback = cb
}

func (t *ServiceMessage) Wait() error {
	if t.Status() != ServiceMessageStatusProcessing {
		return errors.New("can not wait for a non-processing ServiceMessage")
	}
	t.barrier.Wait()
	return nil
}

func (t *ServiceMessage) Response() *Message {
	return t.barrier.Get().(*Message)
}

type ServiceMessageExecutor struct {
	conn           *connection.WRConnection
}

func NewServiceMessageExecutor(c *connection.WRConnection) *ServiceMessageExecutor {
	return &ServiceMessageExecutor{c}
}

func (e *ServiceMessageExecutor) Execute(message *ServiceMessage) {
	// check if messages is processable
	if unProcessableServiceMessageMap[message.Status()] {
		message.resolve(NewErrorMessage(message.Id(), message.From(), message.From(), "request has been cancelled or target server is dead"))
		return
	}
	message.setStatus(ServiceMessageStatusProcessing)
	response, err := e.conn.Request(message.Message)
	if message.Status() == ServiceMessageStatusDead {
		// last check on if messages is killed
		message.resolve(NewErrorMessage(message.Id(), message.From(), message.From(), "request has been cancelled or target server is dead"))
	} else if err != nil {
		message.resolve(NewErrorMessage(message.Id(), message.From(), message.From(), err.Error()))
	} else {
		message.resolve(response)
	}
}
