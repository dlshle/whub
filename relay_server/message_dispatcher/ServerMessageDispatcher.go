package message_dispatcher

import (
	"wsdk/common/logger"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/message_actions"
	"wsdk/relay_common/messages"
	"wsdk/relay_server/container"
	"wsdk/relay_server/context"
	"wsdk/relay_server/controllers/metering"
)

type ServerMessageDispatcher struct {
	dispatcher *message_actions.MessageDispatcher
	metering   metering.IServerMeteringController `$inject:""`
}

func NewServerMessageDispatcher() *ServerMessageDispatcher {
	md := &ServerMessageDispatcher{
		dispatcher: message_actions.NewMessageDispatcher(context.Ctx.Logger().WithPrefix("[MessageDispatcher]")),
	}
	err := container.Container.Fill(md)
	if err != nil {
		panic(err)
	}
	md.init()
	return md
}

func (d *ServerMessageDispatcher) registerHandler(handler message_actions.IMessageHandler) {
	d.dispatcher.RegisterHandler(handler)
}

func (d *ServerMessageDispatcher) logger() *logger.SimpleLogger {
	return d.dispatcher.Logger
}

func (d *ServerMessageDispatcher) init() {
	// register common message handlers
	d.registerHandler(message_actions.NewPingMessageHandler(context.Ctx.Server()))
	d.registerHandler(message_actions.NewInvalidMessageHandler(context.Ctx.Server()))
	d.registerHandler(NewClientDescriptorMessageHandler())
	d.registerHandler(NewServiceRequestMessageHandler())
}

func (d *ServerMessageDispatcher) Dispatch(message *messages.Message, conn connection.IConnection) {
	/*
	 * this function can either be called from a reading loop coroutine or from http handler goroutine, in order to
	 * make the reading loop more effective(less blocking time), actual message dispatching will be done on another
	 * goroutine.
	 * e.g. read -msg-> dispatcher(dispatch and run msg0) -> read -msg1-> dispatcher(...msg1) -> read
	 *                                                           msg0 handle done, write back to conn
	 *                                                                  msg1 handle done, write back to conn
	 */
	if message == nil {
		return
	}
	d.logger().Printf("receive message %s from %s", message.String(), conn.Address())
	d.metering.TraceMessagePerformance(message.Id())
	context.Ctx.AsyncTaskPool().Schedule(func() {
		d.dispatcher.Dispatch(message, conn)
		d.metering.Stop(d.metering.GetAssembledTraceId(metering.TMessagePerformance, message.Id()))
	})
}

func (d *ServerMessageDispatcher) GetHandler(msgType int) message_actions.IMessageHandler {
	return d.dispatcher.GetHandler(msgType)
}
