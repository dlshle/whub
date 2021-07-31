package relay_client

import (
	"wsdk/relay_common/connection"
	"wsdk/relay_common/message_actions"
	"wsdk/relay_common/messages"
	"wsdk/relay_server/context"
)

type ClientMessageDispatcher struct {
	*message_actions.MessageDispatcher
}

func NewClientMessageDispatcher() *ClientMessageDispatcher {
	md := &ClientMessageDispatcher{
		MessageDispatcher: message_actions.NewMessageDispatcher(context.Ctx.Logger().WithPrefix("[MessageDispatcher]")),
	}
	md.init()
	return md
}

func (d *ClientMessageDispatcher) init() {
	// register common message handlers
	d.RegisterHandler(message_actions.NewPingMessageHandler(Ctx.Identity()))
}

func (d *ClientMessageDispatcher) Dispatch(message *messages.Message, conn connection.IConnection) {
	if message == nil {
		return
	}
	d.Logger.Printf("receive message %s from %s", message.String(), conn.Address())
	Ctx.AsyncTaskPool().Schedule(func() {
		d.MessageDispatcher.Dispatch(message, conn)
	})
}
