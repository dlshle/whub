package message_dispatcher

import (
	"fmt"
	"sync"
	"wsdk/common/logger"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/message_actions"
	"wsdk/relay_common/messages"
	"wsdk/relay_server/context"
)

type ServerMessageDispatcher struct {
	handlers map[int]message_actions.IMessageHandler
	logger   *logger.SimpleLogger
	lock     *sync.RWMutex
}

func NewServerMessageDispatcher() *ServerMessageDispatcher {
	return &ServerMessageDispatcher{
		handlers: make(map[int]message_actions.IMessageHandler),
		logger:   context.Ctx.Logger().WithPrefix("[ServerMessageDispatcher]"),
		lock:     new(sync.RWMutex),
	}
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
	d.RegisterHandler(NewServiceRequestMessageHandler())
}

func (d *ServerMessageDispatcher) RegisterHandler(handler message_actions.IMessageHandler) {
	d.withWrite(func() {
		d.handlers[handler.Type()] = handler
	})
}

func (d *ServerMessageDispatcher) Dispatch(message *messages.Message, conn connection.IConnection) {
	d.logger.Printf("receive message %s", message.String())
	context.Ctx.AsyncTaskPool().Schedule(func() {
		handler := d.handlers[message.MessageType()]
		if handler == nil {
			handler = d.handlers[messages.MessageTypeUnknown]
			return
		}
		err := handler.Handle(message, conn)
		if err != nil {
			// TODO do something
			fmt.Println("handler error ", err)
		}
	})
}
