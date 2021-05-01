package WRCommon

import (
	"errors"
	"sync"
)

func init() {
	initTrackableMessage()
}

// Message Protocol
const (
	MessageProtocolSimple     = 0 // use json string
	MessageProtocolFlatBuffer = 1 // use FlatBuffer
)

// Message Type
const (
	MessageTypeProtocolUpdate = -1
	MessageTypePing           = 0
	MessageTypePong           = 1
	MessageTypeACK            = 2
	MessageTypeText           = 3
	MessageTypeStream         = 4
	MessageTypeJSON           = 5
)

type Message struct {
	id          string
	from        string // use id or credential here
	to          string // use id or credential here
	messageType int
	payload     []byte
}

func NewMessage(id string, from string, to string, messageType int, payload []byte) *Message {
	return &Message{id, from, to, messageType, payload}
}

const (
	TrackableMessageStatusQueued     = 0
	TrackableMessageStatusProcessing = 1
	TrackableMessageStatusDead       = 2 // when health check failed
	TrackableMessageStatusFinished   = 3
	TrackableMessageStatusCancelled  = 4
)

var statusCodeStringMap map[int]string

func initTrackableMessage() {
	statusCodeStringMap = make(map[int]string)
	statusCodeStringMap[TrackableMessageStatusQueued] = "queued"
	statusCodeStringMap[TrackableMessageStatusProcessing] = "processing"
	statusCodeStringMap[TrackableMessageStatusDead] = "dead"
	statusCodeStringMap[TrackableMessageStatusFinished] = "finished"
	statusCodeStringMap[TrackableMessageStatusCancelled] = "cancelled"
}

type ServiceMessage struct {
	channel chan *Message
	status  int
	lock    *sync.RWMutex
	*Message
	onStatusChangeCallback func(int)
}

func NewTrackableMessage(m *Message) *ServiceMessage {
	return &ServiceMessage{make(chan *Message), TrackableMessageStatusQueued, new(sync.RWMutex), m, nil}
}

type IServiceMessage interface {
	Status() int
	Kill() error
	Cancel() error
	IsDead() bool
	IsCancelled() bool
	IsFinished() bool
	OnStatusChange(func(int))
	Resolve(*Message) error
	Wait() error // wait for the state to transit to final (dead/finished/cancelled)
}

func (t *ServiceMessage) withWrite(cb func()) {
	t.lock.Lock()
	defer t.lock.Unlock()
	cb()
}

func (t *ServiceMessage) setStatus(status int) {
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
		t.status = TrackableMessageStatusDead
		close(t.channel)
	})
	return nil
}

func (t *ServiceMessage) Cancel() error {
	if t.Status() != TrackableMessageStatusQueued {
		return errors.New("unable to cancel a " + statusCodeStringMap[t.Status()] + " ServiceMessage")
	}
	t.withWrite(func() {
		t.status = TrackableMessageStatusCancelled
		close(t.channel)
	})
	return nil
}

func (t *ServiceMessage) Resolve(m *Message) error {
	if t.Status() != TrackableMessageStatusProcessing {
		return errors.New("can not resolve a non-processing ServiceMessage")
	}
	t.withWrite(func() {
		t.status = TrackableMessageStatusFinished
		t.channel <- m
	})
	return nil
}

func (t *ServiceMessage) IsDead() bool {
	return t.Status() == TrackableMessageStatusDead
}

func (t *ServiceMessage) IsCancelled() bool {
	return t.Status() == TrackableMessageStatusCancelled
}

func (t *ServiceMessage) IsFinished() bool {
	return t.Status() == TrackableMessageStatusFinished
}

func (t *ServiceMessage) OnStatusChange(cb func(int)) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.onStatusChangeCallback = cb
}

func (t *ServiceMessage) Wait() error {
	if t.Status() != TrackableMessageStatusProcessing {
		return errors.New("can not wait for a non-processing ServiceMessage")
	}
	<-t.channel
	return nil
}
