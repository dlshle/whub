package WRCommon

import (
	"errors"
	"reflect"
	"sync"
	"sync/atomic"
)

const DefaultMaxListeners = 256

type MessageListener func(*Message)
type Disposable func()

type MessageNotificationEmitter struct {
	ap map[string][]MessageListener
	lock *sync.RWMutex
	maxNumOfMessageListeners int
}

type IMessageNotificationEmitter interface {
	HasEvent(eventId string) bool
	MessageListenerCount(eventId string) int
	Notify(eventId string, payload *Message)
	On(eventId string, listener MessageListener) (Disposable, error)
	Once(eventId string, listener MessageListener) (Disposable, error)
	Off(eventId string, listener MessageListener)
	OffAll(eventId string)
}

func New(maxMessageListenerCount int) IMessageNotificationEmitter {
	if maxMessageListenerCount < 1 || maxMessageListenerCount > DefaultMaxListeners {
		maxMessageListenerCount = DefaultMaxListeners
	}
	return &MessageNotificationEmitter{make(map[string][]MessageListener), new(sync.RWMutex), maxMessageListenerCount}
}

func (e *MessageNotificationEmitter) withWrite(cb func()) {
	e.lock.Lock()
	defer e.lock.Unlock()
	cb()
}

func (e *MessageNotificationEmitter) getMessageListeners(eventId string) []MessageListener {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return e.ap[eventId]
}

func (e *MessageNotificationEmitter) addMessageListener(eventId string, listener MessageListener) (err error) {
	e.withWrite(func() {
		listeners := e.ap[eventId]
		if listeners == nil {
			listeners = make([]MessageListener, 0, e.maxNumOfMessageListeners)
		} else if len(listeners) >= e.maxNumOfMessageListeners {
			err = errors.New("listener count exceeded maxMessageListenerCount for event " +
				eventId +
				", please use SetMaxMessageListenerCount to top maxMessageListenerCount.")
			return
		}
		e.ap[eventId] = append(listeners, listener)
	})
	return
}

func (e *MessageNotificationEmitter) indexOfMessageListener(eventId string, listener MessageListener) int {
	listenerPtr := reflect.ValueOf(listener).Pointer()
	e.lock.RLock()
	defer e.lock.RUnlock()
	if e.ap[eventId] == nil {
		return -1
	}
	for i, f := range e.ap[eventId] {
		currPtr := reflect.ValueOf(f).Pointer()
		if listenerPtr == currPtr {
			return i
		}
	}
	return -1
}

func (e *MessageNotificationEmitter) removeIthMessageListener(eventId string, listenerIdx int) {
	if e.MessageListenerCount(eventId) < listenerIdx {
		return
	}
	e.withWrite(func() {
		allMessageListeners := e.ap[eventId]
		e.ap[eventId] = append(allMessageListeners[:listenerIdx], allMessageListeners[listenerIdx+1:]...)
	})
}

func (e *MessageNotificationEmitter) HasEvent(eventId string) bool {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return e.ap[eventId] != nil
}

func (e *MessageNotificationEmitter) Notify(eventId string, payload *Message) {
	if !e.HasEvent(eventId) {
		return
	}
	e.lock.RLock()
	listeners := e.ap[eventId]
	e.lock.RUnlock()
	// defer e.lock.RUnlock()
	var wg sync.WaitGroup
	for _, f := range listeners {
		if f != nil {
			wg.Add(1)
			go func(listener MessageListener) {
				listener(payload)
				wg.Done()
			}(f)
		}
	}
	wg.Wait()
}

func (e *MessageNotificationEmitter) MessageListenerCount(eventId string) int {
	e.lock.RLock()
	defer e.lock.RUnlock()
	if e.ap[eventId] == nil {
		return 0
	}
	return len(e.ap[eventId])
}

func (e *MessageNotificationEmitter) On(eventId string, listener MessageListener) (Disposable, error) {
	err := e.addMessageListener(eventId, listener)
	if err != nil {
		return nil, err
	}
	return func() {
		e.Off(eventId, listener)
	}, nil
}

func (e *MessageNotificationEmitter) Once(eventId string, listener MessageListener) (Disposable, error) {
	hasFired := atomic.Value{}
	hasFired.Store(false)
	// need this to refer from the actualMessageListener
	var actualMessageListenerPtr func(*Message)
	actualMessageListener := func(param *Message) {
		if hasFired.Load().(bool) {
			e.Off(eventId, actualMessageListenerPtr)
			return
		}
		listener(param)
		e.Off(eventId, actualMessageListenerPtr)
		hasFired.Store(true)
	}
	actualMessageListenerPtr = actualMessageListener
	err := e.addMessageListener(eventId, actualMessageListenerPtr)
	if err != nil {
		return nil, err
	}
	return func() {
		e.Off(eventId, actualMessageListenerPtr)
		// manually free two pointers
		actualMessageListenerPtr = nil
		actualMessageListener = nil
	}, nil
}

func (e *MessageNotificationEmitter) Off(eventId string, listener MessageListener) {
	if !e.HasEvent(eventId) {
		return
	}
	listenerIdx := e.indexOfMessageListener(eventId, listener)
	e.removeIthMessageListener(eventId, listenerIdx)
}

func (e *MessageNotificationEmitter) OffAll(eventId string) {
	if !e.HasEvent(eventId) {
		return
	}
	e.withWrite(func() {
		e.ap[eventId] = nil
	})
}