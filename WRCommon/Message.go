package WRCommon

import (
	"errors"
	"fmt"
	"github.com/dlshle/gommon/async"
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
	MessageTypeError          = 6
)

type Message struct {
	id          string
	from        string // use id or credential here
	to          string // use id or credential here
	messageType int
	payload     []byte
}

type IMessage interface {
	Id() string
	From() string
	To() string
	MessageType() int
	Payload() []byte
	String() string
}

func (t *Message) Id() string {
	return t.id
}

func (t *Message) From() string {
	return t.from
}

func (t *Message) To() string {
	return t.to
}

func (t *Message) MessageType() int {
	return t.messageType
}

func (t *Message) Payload() []byte {
	return t.payload
}

func (t *Message) String() string {
	return fmt.Sprintf("{from: \"%s\", to: \"%s\", messageType: %d, payload: %s}", t.from, t.to, t.messageType, t.payload)
}


func NewMessage(id string, from string, to string, messageType int, payload []byte) *Message {
	return &Message{id, from, to, messageType, payload}
}

func NewErrorMessage(id string, from string, to string, errorMessage string) *Message {
	return &Message{id, from, to, MessageTypeError, ([]byte)(errorMessage)}
}

const (
	TrackableMessageStatusQueued     = 0
	TrackableMessageStatusProcessing = 1
	TrackableMessageStatusDead       = 2 // when health check failed
	TrackableMessageStatusFinished   = 3
	TrackableMessageStatusCancelled  = 4
)

var unprocessableServiceMessageMap map[int]bool
var statusCodeStringMap map[int]string

func initTrackableMessage() {
	statusCodeStringMap = make(map[int]string)
	statusCodeStringMap[TrackableMessageStatusQueued] = "queued"
	statusCodeStringMap[TrackableMessageStatusProcessing] = "processing"
	statusCodeStringMap[TrackableMessageStatusDead] = "dead"
	statusCodeStringMap[TrackableMessageStatusFinished] = "finished"
	statusCodeStringMap[TrackableMessageStatusCancelled] = "cancelled"

	unprocessableServiceMessageMap = make(map[int]bool)
	unprocessableServiceMessageMap[TrackableMessageStatusDead] = true
	unprocessableServiceMessageMap[TrackableMessageStatusCancelled] = true
}

type ServiceMessage struct {
	barrier *async.StatefulBarrier
	status  int
	lock    *sync.RWMutex
	*Message
	onStatusChangeCallback func(int)
}

func NewServiceMessage(m *Message) *ServiceMessage {
	return &ServiceMessage{async.NewStatefulBarrier(), TrackableMessageStatusQueued, new(sync.RWMutex), m, nil}
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
		t.barrier.OpenWith(nil)
	})
	return nil
}

func (t *ServiceMessage) Cancel() error {
	if t.Status() != TrackableMessageStatusQueued {
		return errors.New("unable to cancel a " + statusCodeStringMap[t.Status()] + " ServiceMessage")
	}
	t.withWrite(func() {
		t.status = TrackableMessageStatusCancelled
		t.barrier.OpenWith(nil)
	})
	return nil
}

func (t *ServiceMessage) resolve(m *Message) error {
	if t.Status() != TrackableMessageStatusProcessing {
		return errors.New("can not resolve a non-processing ServiceMessage")
	}
	t.withWrite(func() {
		t.status = TrackableMessageStatusFinished
		t.barrier.OpenWith(m)
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
	t.barrier.Wait()
	return nil
}

func (t *ServiceMessage) Response() *Message {
	return t.barrier.Get().(*Message)
}

type ServiceMessageExecutor struct {
	conn           *WRConnection
}

func NewServiceMessageExecutor(c *WRConnection) *ServiceMessageExecutor {
	return &ServiceMessageExecutor{c}
}

// dispatcher should make sure the message is to the right receiver
func (e *ServiceMessageExecutor) Execute(message *ServiceMessage) {
	// check if message is processable
	if unprocessableServiceMessageMap[message.Status()] {
		message.resolve(NewErrorMessage(message.Id(), message.From(), message.From(), "request has been cancelled or target server is dead"))
		return
	}
	message.setStatus(TrackableMessageStatusProcessing)
	response, err := e.conn.Request(message.Message)
	if err != nil {
		message.resolve(NewErrorMessage(message.Id(), message.From(), message.From(), err.Error()))
	} else {
		message.resolve(response)
	}
}
