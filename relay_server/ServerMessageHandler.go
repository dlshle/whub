package relay_server

import (
	"errors"
	"reflect"
	"sync"
	"wsdk/relay_common"
	"wsdk/relay_common/messages"
)

const (
	HandlerPriorityServiceRequest = 2
	HandlerPriorityClientMessage  = 3
	HandlerPriorityInvalidMessage = 99 // always put this on the last
)

type IServerMessageHandler interface {
	messages.IMessageHandler
	Priority() int
}

type ServerMessageHandlerManager struct {
	handlers []IServerMessageHandler
	lock     *sync.RWMutex
}

type IServerMessageHandlerManager interface {
	messages.IMessageHandler
	RegisterHandler(handler IServerMessageHandler)
	UnregisterHandler(handler IServerMessageHandler)
	composeHandlers() messages.NextMessageHandler
}

func (m *ServerMessageHandlerManager) withWrite(cb func()) {
	m.lock.Lock()
	defer m.lock.Unlock()
	cb()
}

func (m *ServerMessageHandlerManager) hasHandler(handler IServerMessageHandler) bool {
	handlerPtr := reflect.ValueOf(handler).Pointer()
	m.lock.RLock()
	defer m.lock.RUnlock()
	for _, h := range m.handlers {
		currPtr := reflect.ValueOf(h).Pointer()
		if handlerPtr == currPtr {
			return true
		}
	}
	return false
}

func (m *ServerMessageHandlerManager) Handle(message *messages.Message, next messages.NextMessageHandler) (*messages.Message, error) {
	return m.composeHandlers()(message, next)
}

// a bad imitation of koa-compose
func (m *ServerMessageHandlerManager) composeHandlers() messages.HandlerFunction {
	return func(message *messages.Message, next messages.NextMessageHandler) (*messages.Message, error) {
		index := -1
		makeNextHandlerFn := func(i int) (*messages.Message, error) {
			/*
				  if (i <= index) return Promise.reject(new Error('next() called multiple times'))
				  index = i
				  let handler = middleware[i]
				  if (i === middleware.length) handler = next
				  if (!handler) return Promise.resolve()
				  try {
					return Promise.resolve(handler(context, dispatch.bind(null, i + 1)));
				  } catch (err) {
					return Promise.reject(err)
				  }
			*/
			if i <= index {
				return nil, errors.New("next() called many times")
			}
			index = i
			handler := m.handlers[i]
			shouldUseNext := false
			if i == len(m.handlers) {
				shouldUseNext = true
			}
			if handler == nil || shouldUseNext && next == nil {
				return nil, nil
			}
			if shouldUseNext {
				return next(message)
			} else {
				return handler.Handle(message, next)
			}
		}
		return makeNextHandlerFn(0)
	}
}

func (m *ServerMessageHandlerManager) RegisterHandler(handler IServerMessageHandler) {
	if m.hasHandler(handler) {
		return
	}
	m.withWrite(func() {
		// TODO bin search find pos and insert
	})
}

func (m *ServerMessageHandlerManager) UnregisterHandler(handler IServerMessageHandler) {
	// TODO bin search find pos and remove
}

type InvalidMessageHandler struct {
	ctx *relay_common.WRContext
}

func (h *InvalidMessageHandler) Handle(message *messages.Message) *messages.Message {
	return messages.NewErrorMessage(message.Id(), h.ctx.Identity().Id(), message.From(), message.Uri(), NewInvalidMessageError().Json())
}

func (h *InvalidMessageHandler) Priority() int {
	return HandlerPriorityInvalidMessage
}
