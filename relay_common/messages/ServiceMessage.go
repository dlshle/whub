package messages

import (
	"errors"
	"sync"
	"wsdk/gommon/async"
	"wsdk/relay_common/connection"
)

func init() {
	initServiceRequest()
}

const (
	ServiceRequestStatusQueued     = 0
	ServiceRequestStatusProcessing = 1
	ServiceRequestStatusDead       = 2 // when health check failed
	ServiceRequestStatusFinished   = 3
	ServiceRequestStatusCancelled  = 4
)

var unProcessableServiceRequestMap map[int]bool
var statusCodeStringMap map[int]string

func initServiceRequest() {
	statusCodeStringMap = make(map[int]string)
	statusCodeStringMap[ServiceRequestStatusQueued] = "queued"
	statusCodeStringMap[ServiceRequestStatusProcessing] = "processing"
	statusCodeStringMap[ServiceRequestStatusDead] = "dead"
	statusCodeStringMap[ServiceRequestStatusFinished] = "finished"
	statusCodeStringMap[ServiceRequestStatusCancelled] = "cancelled"

	unProcessableServiceRequestMap = make(map[int]bool)
	unProcessableServiceRequestMap[ServiceRequestStatusDead] = true
	unProcessableServiceRequestMap[ServiceRequestStatusCancelled] = true
}

type ServiceRequest struct {
	barrier *async.StatefulBarrier
	status  int
	lock    *sync.RWMutex
	*Message
	onStatusChangeCallback func(int)
}

func NewServiceRequest(m *Message) *ServiceRequest {
	return &ServiceRequest{async.NewStatefulBarrier(), ServiceRequestStatusQueued, new(sync.RWMutex), m, nil}
}

type IServiceRequest interface {
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

func (t *ServiceRequest) withWrite(cb func()) {
	t.lock.Lock()
	defer t.lock.Unlock()
	cb()
}

func (t *ServiceRequest) setStatus(status int) {
	if t.Status() == ServiceRequestStatusFinished {
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

func (t *ServiceRequest) Status() int {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.status
}

func (t *ServiceRequest) Kill() error {
	if t.Status() > 1 {
		return errors.New("unable to kill a " + statusCodeStringMap[t.Status()] + " ServiceRequest")
	}
	t.withWrite(func() {
		t.status = ServiceRequestStatusDead
		t.barrier.OpenWith(nil)
	})
	return nil
}

func (t *ServiceRequest) Cancel() error {
	if t.Status() > 1 {
		return errors.New("unable to cancel a " + statusCodeStringMap[t.Status()] + " ServiceRequest")
	}
	t.withWrite(func() {
		t.status = ServiceRequestStatusCancelled
		t.barrier.OpenWith(nil)
	})
	return nil
}

func (t *ServiceRequest) resolve(m *Message) error {
	if t.Status() != ServiceRequestStatusProcessing {
		return errors.New("can not resolve a non-processing ServiceRequest")
	}
	t.withWrite(func() {
		t.status = ServiceRequestStatusFinished
		t.barrier.OpenWith(m)
	})
	return nil
}

func (t *ServiceRequest) IsDead() bool {
	return t.Status() == ServiceRequestStatusDead
}

func (t *ServiceRequest) IsCancelled() bool {
	return t.Status() == ServiceRequestStatusCancelled
}

func (t *ServiceRequest) IsFinished() bool {
	return t.Status() == ServiceRequestStatusFinished
}

func (t *ServiceRequest) OnStatusChange(cb func(int)) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.onStatusChangeCallback = cb
}

func (t *ServiceRequest) Wait() error {
	if t.Status() != ServiceRequestStatusProcessing {
		return errors.New("can not wait for a non-processing ServiceRequest")
	}
	t.barrier.Wait()
	return nil
}

func (t *ServiceRequest) Response() *Message {
	return t.barrier.Get().(*Message)
}

type ServiceRequestExecutor struct {
	conn *connection.WRConnection
}

func NewServiceRequestExecutor(c *connection.WRConnection) *ServiceRequestExecutor {
	return &ServiceRequestExecutor{c}
}

func (e *ServiceRequestExecutor) Execute(message *ServiceRequest) {
	// check if messages is processable
	if unProcessableServiceRequestMap[message.Status()] {
		message.resolve(NewErrorMessage(message.Id(), message.From(), message.From(), message.Uri(), "request has been cancelled or target server is dead"))
		return
	}
	message.setStatus(ServiceRequestStatusProcessing)
	response, err := e.conn.Request(message.Message)
	if message.Status() == ServiceRequestStatusDead {
		// last check on if messages is killed
		message.resolve(NewErrorMessage(message.Id(), message.From(), message.From(), message.Uri(), "request has been cancelled or target server is dead"))
	} else if err != nil {
		message.resolve(NewErrorMessage(message.Id(), message.From(), message.From(), message.Uri(), err.Error()))
	} else {
		message.resolve(response)
	}
}
