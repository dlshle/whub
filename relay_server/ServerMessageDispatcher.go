package relay_server

import (
	"sync"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/messages"
)

type ServerMessageDispatcher struct {
	ctx      *Context
	handlers map[int]messages.IMessageHandler
	lock     *sync.RWMutex
}

func (d *ServerMessageDispatcher) withWrite(cb func()) {
	d.lock.Lock()
	defer d.lock.Unlock()
	cb()
}

func (d *ServerMessageDispatcher) init() {
	// register common message handlers
	d.RegisterHandler(messages.NewPingMessageHandler(d.ctx.Identity()), true)
	d.RegisterHandler(messages.NewInvalidMessageHandler(d.ctx.Identity()), true)
	// TODO how to register ServiceUpdateNotificationMessageHandler
}

func (d *ServerMessageDispatcher) RegisterHandler(handler messages.IMessageHandler, override bool) {
	h := d.handlers[handler.Type()]
	d.withWrite(func() {
		if h != nil && override {
			d.handlers[handler.Type()] = handler
		} else {
			// don't know what to do...
		}
	})
}

func (d *ServerMessageDispatcher) Dispatch(message *messages.Message, conn *connection.WRConnection) {
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
