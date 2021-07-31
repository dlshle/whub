package message_actions

import (
	"sync"
	"wsdk/common/logger"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/messages"
)

type IMessageDispatcher interface {
	Dispatch(message *messages.Message, conn connection.IConnection)
}

type MessageDispatcher struct {
	handlers map[int]IMessageHandler
	Logger   *logger.SimpleLogger
	lock     *sync.RWMutex
}

func NewMessageDispatcher(logger *logger.SimpleLogger) *MessageDispatcher {
	md := &MessageDispatcher{
		handlers: make(map[int]IMessageHandler),
		Logger:   logger,
		lock:     new(sync.RWMutex),
	}
	return md
}

func (d *MessageDispatcher) withWrite(cb func()) {
	d.lock.Lock()
	defer d.lock.Unlock()
	cb()
}

func (d *MessageDispatcher) RegisterHandler(handler IMessageHandler) {
	d.Logger.Printf("handler for message type %d has been registered", handler.Type())
	d.withWrite(func() {
		d.handlers[handler.Type()] = handler
	})
}

func (d *MessageDispatcher) UnregisterHandler(msgType int) (success bool) {
	d.Logger.Printf("handler for message type %d has been registered", msgType)
	success = false
	d.withWrite(func() {
		if d.handlers[msgType] != nil {
			success = true
			delete(d.handlers, msgType)
		}
	})
	return
}

func (d *MessageDispatcher) Dispatch(message *messages.Message, conn connection.IConnection) {
	handler := d.handlers[message.MessageType()]
	if handler == nil {
		d.Logger.Println("can not find handler for message: ", message.String())
		return
	}
	err := handler.Handle(message, conn)
	if err != nil {
		d.Logger.Printf("message %s handler error due to %s", message.String(), err.Error())
	}
}

func (d *MessageDispatcher) GetHandler(msgType int) IMessageHandler {
	return d.handlers[msgType]
}
