package relay_client

import (
	"wsdk/relay_client/container"
	context2 "wsdk/relay_client/context"
	"wsdk/relay_client/controllers"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/message_actions"
	"wsdk/relay_common/messages"
	"wsdk/relay_server/context"
)

type ClientMessageDispatcher struct {
	*message_actions.MessageDispatcher
	m controllers.IClientMeteringController `$inject:""`
}

func NewClientMessageDispatcher() *ClientMessageDispatcher {
	md := &ClientMessageDispatcher{
		MessageDispatcher: message_actions.NewMessageDispatcher(context.Ctx.Logger().WithPrefix("[MessageDispatcher]")),
	}
	err := container.Container.Fill(md)
	if err != nil {
		panic(err)
	}
	md.init()
	return md
}

func (d *ClientMessageDispatcher) init() {
	// register common message handlers
	d.RegisterHandler(message_actions.NewPingMessageHandler(context2.Ctx.Identity()))
}

func (d *ClientMessageDispatcher) Dispatch(message *messages.Message, conn connection.IConnection) {
	if message == nil {
		return
	}
	d.Logger.Printf("receive message %s from %s", message.String(), conn.Address())
	d.m.TraceMessagePerformance(message.Id())
	context2.Ctx.AsyncTaskPool().Schedule(func() {
		d.MessageDispatcher.Dispatch(message, conn)
		d.m.Stop(d.m.GetAssembledTraceId(controllers.TMessagePerformance, message.Id()))
	})
}
