package relay_client

import (
	"wsdk/relay_client/container"
	"wsdk/relay_client/context"
	"wsdk/relay_client/controllers"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/dispatcher"
	"wsdk/relay_common/messages"
)

type ClientMessageDispatcher struct {
	*dispatcher.MessageDispatcher
	m controllers.IClientMeteringController `$inject:""`
}

func NewClientMessageDispatcher() *ClientMessageDispatcher {
	md := &ClientMessageDispatcher{
		MessageDispatcher: dispatcher.NewMessageDispatcher(context.Ctx.Logger().WithPrefix("[MessageDispatcher]")),
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
	d.RegisterHandler(dispatcher.NewPingMessageHandler(context.Ctx.Identity()))
}

func (d *ClientMessageDispatcher) Dispatch(message messages.IMessage, conn connection.IConnection) {
	if message == nil {
		return
	}
	d.Logger.Printf("receive message %s from %s", message.String(), conn.Address())
	d.m.TraceMessagePerformance(message.Id())
	context.Ctx.AsyncTaskPool().Schedule(func() {
		d.MessageDispatcher.Dispatch(message, conn)
		d.m.Stop(d.m.GetAssembledTraceId(controllers.TMessagePerformance, message.Id()))
		message.Dispose()
	})
}
