package message_dispatcher

import (
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
	md := &ServerMessageDispatcher{
		handlers: make(map[int]message_actions.IMessageHandler),
		logger:   context.Ctx.Logger().WithPrefix("[ServerMessageDispatcher]"),
		lock:     new(sync.RWMutex),
	}
	md.init()
	return md
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
	d.RegisterHandler(NewClientDescriptorMessageHandler())
	d.RegisterHandler(NewServiceRequestMessageHandler())
}

func (d *ServerMessageDispatcher) RegisterHandler(handler message_actions.IMessageHandler) {
	d.logger.Printf("handler for message type %d has been registered", handler.Type())
	d.withWrite(func() {
		d.handlers[handler.Type()] = handler
	})
}

func (d *ServerMessageDispatcher) Dispatch(message *messages.Message, conn connection.IConnection) {
	if message == nil {
		return
	}
	d.logger.Printf("receive message %s from %s", message.String(), conn.Address())
	context.Ctx.AsyncTaskPool().Schedule(func() {
		handler := d.handlers[message.MessageType()]
		if handler == nil {
			d.logger.Println("can not find handler for message: ", message.String(), " will use respond with invalid message error")
			handler = d.handlers[messages.MessageTypeUnknown]
		}
		err := handler.Handle(message, conn)
		if err != nil {
			d.logger.Printf("message %s handler error due to %s", message.String(), err.Error())
		}
	})
}

func (d *ServerMessageDispatcher) GetHandler(msgType int) message_actions.IMessageHandler {
	return d.handlers[msgType]
}
