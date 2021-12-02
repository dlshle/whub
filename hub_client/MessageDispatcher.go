package hub_client

import (
	"whub/hub_client/container"
	"whub/hub_client/context"
	"whub/hub_client/controllers"
	"whub/hub_common/connection"
	"whub/hub_common/dispatcher"
	"whub/hub_common/messages"
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
