package relay_server

import (
	"sync"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/message_actions"
	"wsdk/relay_common/messages"
	"wsdk/relay_server/context"
)

type ServerMessageDispatcher struct {
	ctx      *context.Context
	handlers map[int]message_actions.IMessageHandler
	lock     *sync.RWMutex
}

func (d *ServerMessageDispatcher) withWrite(cb func()) {
	d.lock.Lock()
	defer d.lock.Unlock()
	cb()
}

func (d *ServerMessageDispatcher) init() {
	// register common message handlers
	d.RegisterHandler(message_actions.NewPingMessageHandler(d.ctx.Server()), true)
	d.RegisterHandler(message_actions.NewInvalidMessageHandler(d.ctx.Server()), true)
	// TODO how to register ServiceUpdateNotificationMessageHandler
}

func (d *ServerMessageDispatcher) RegisterHandler(handler message_actions.IMessageHandler, override bool) {
	h := d.handlers[handler.Type()]
	d.withWrite(func() {
		if h != nil && override {
			d.handlers[handler.Type()] = handler
		} else {
			// don't know what to do...
		}
	})
}

func (d *ServerMessageDispatcher) Dispatch(message *messages.Message, conn *connection.Connection) {
	d.ctx.AsyncTaskPool().Schedule(func() {
		handler := d.handlers[message.MessageType()]
		if handler == nil {
			handler = d.handlers[messages.MessageTypeUnknown]
		}
		err := handler.Handle(message, conn)
		if err != nil {
			// TODO do something
		}
	})
}
