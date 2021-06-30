package messages

import (
	"fmt"
	"sync"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/message_actions"
	"wsdk/relay_common/messages"
	"wsdk/relay_server/context"
	"wsdk/relay_server/handlers"
)

type ServerMessageDispatcher struct {
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
	d.RegisterHandler(message_actions.NewPingMessageHandler(context.Ctx.Server()))
	d.RegisterHandler(message_actions.NewInvalidMessageHandler(context.Ctx.Server()))
	d.RegisterHandler(handlers.NewServiceNotificationMessageHandler())
	d.RegisterHandler(handlers.NewServiceRequestMessageHandler())
}

func (d *ServerMessageDispatcher) RegisterHandler(handler message_actions.IMessageHandler) {
	d.withWrite(func() {
		d.handlers[handler.Type()] = handler
	})
}

func (d *ServerMessageDispatcher) Dispatch(message *messages.Message, conn *connection.Connection) {
	context.Ctx.AsyncTaskPool().Schedule(func() {
		handler := d.handlers[message.MessageType()]
		if handler == nil {
			handler = d.handlers[messages.MessageTypeUnknown]
		}
		err := handler.Handle(message, conn)
		if err != nil {
			// TODO do something
			fmt.Println("handler error ", err)
		}
	})
}
