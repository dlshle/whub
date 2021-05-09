package notification

import (
	"errors"
	"reflect"
	"sync"
	"sync/atomic"
	"wsdk/relay_common/messages"
)

const DefaultMaxListeners = 256

type MessageListener func(*messages.Message)
type Disposable func()

type WRNotificationEmitter struct {
	ap map[string][]MessageListener
	lock *sync.RWMutex
	maxNumOfMessageListeners int
}

type IWRNotificationEmitter interface {
	HasEvent(eventId string) bool
	MessageListenerCount(eventId string) int
	Notify(eventId string, payload *messages.Message)
	On(eventId string, listener MessageListener) (Disposable, error)
	Once(eventId string, listener MessageListener) (Disposable, error)
	Off(eventId string, listener MessageListener)
	OffAll(eventId string)
}

func New(maxMessageListenerCount int) IWRNotificationEmitter {
	if maxMessageListenerCount < 1 || maxMessageListenerCount > DefaultMaxListeners {
		maxMessageListenerCount = DefaultMaxListeners
	}
	return &WRNotificationEmitter{make(map[string][]MessageListener), new(sync.RWMutex), maxMessageListenerCount}
}

func (e *WRNotificationEmitter) withWrite(cb func()) {
	e.lock.Lock()
	defer e.lock.Unlock()
	cb()
}

func (e *WRNotificationEmitter) getMessageListeners(eventId string) []MessageListener {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return e.ap[eventId]
}

func (e *WRNotificationEmitter) addMessageListener(eventId string, listener MessageListener) (err error) {
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

func (e *WRNotificationEmitter) indexOfMessageListener(eventId string, listener MessageListener) int {
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

func (e *WRNotificationEmitter) removeIthMessageListener(eventId string, listenerIdx int) {
	if e.MessageListenerCount(eventId) < listenerIdx {
		return
	}
	e.withWrite(func() {
		allMessageListeners := e.ap[eventId]
		e.ap[eventId] = append(allMessageListeners[:listenerIdx], allMessageListeners[listenerIdx+1:]...)
	})
}

func (e *WRNotificationEmitter) HasEvent(eventId string) bool {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return e.ap[eventId] != nil
}

func (e *WRNotificationEmitter) Notify(eventId string, payload *messages.Message) {
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

func (e *WRNotificationEmitter) MessageListenerCount(eventId string) int {
	e.lock.RLock()
	defer e.lock.RUnlock()
	if e.ap[eventId] == nil {
		return 0
	}
	return len(e.ap[eventId])
}

func (e *WRNotificationEmitter) On(eventId string, listener MessageListener) (Disposable, error) {
	err := e.addMessageListener(eventId, listener)
	if err != nil {
		return nil, err
	}
	return func() {
		e.Off(eventId, listener)
	}, nil
}

func (e *WRNotificationEmitter) Once(eventId string, listener MessageListener) (Disposable, error) {
	hasFired := atomic.Value{}
	hasFired.Store(false)
	// need this to refer from the actualMessageListener
	var actualMessageListenerPtr func(*messages.Message)
	actualMessageListener := func(param *messages.Message) {
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

func (e *WRNotificationEmitter) Off(eventId string, listener MessageListener) {
	if !e.HasEvent(eventId) {
		return
	}
	listenerIdx := e.indexOfMessageListener(eventId, listener)
	e.removeIthMessageListener(eventId, listenerIdx)
}

func (e *WRNotificationEmitter) OffAll(eventId string) {
	if !e.HasEvent(eventId) {
		return
	}
	e.withWrite(func() {
		e.ap[eventId] = nil
	})
}