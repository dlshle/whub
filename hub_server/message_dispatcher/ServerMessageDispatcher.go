package message_dispatcher

import (
	"whub/common/logger"
	"whub/hub_common/connection"
	"whub/hub_common/dispatcher"
	"whub/hub_common/messages"
	"whub/hub_server/context"
	"whub/hub_server/module_base"
	"whub/hub_server/modules/metering"
)

type ServerMessageDispatcher struct {
	dispatcher *dispatcher.MessageDispatcher
	metering   metering.IMeteringModule `module:""`
}

func NewServerMessageDispatcher() *ServerMessageDispatcher {
	md := &ServerMessageDispatcher{
		dispatcher: dispatcher.NewMessageDispatcher(context.Ctx.Logger().WithPrefix("[MessageDispatcher]")),
	}
	err := module_base.Manager.AutoFill(md)
	if err != nil {
		panic(err)
	}
	md.init()
	return md
}

func (d *ServerMessageDispatcher) registerHandler(handler dispatcher.IMessageHandler) {
	d.dispatcher.RegisterHandler(handler)
}

func (d *ServerMessageDispatcher) logger() *logger.SimpleLogger {
	return d.dispatcher.Logger
}

func (d *ServerMessageDispatcher) init() {
	// register common message handlers
	d.registerHandler(dispatcher.NewPingMessageHandler(context.Ctx.Server()))
	d.registerHandler(dispatcher.NewInvalidMessageHandler(context.Ctx.Server()))
	d.registerHandler(NewServiceRequestMessageHandler())
}

func (d *ServerMessageDispatcher) Dispatch(message messages.IMessage, conn connection.IConnection) {
	/*
	 * This function can either be called from a reading loop coroutine or from http handler goroutine. In order to
	 * make the reading loop more effective(less read blocking), actual message dispatching will be done on another
	 * goroutine.
	 * e.g. read -msg-> dispatcher(dispatch and run msg0) -> read -msg1-> dispatcher(...msg1) -> read
	 *                                                           msg0 handle done, write back to conn
	 *                                                                  msg1 handle done, write back to conn
	 */
	if message == nil {
		return
	}
	d.logger().Printf("receive message %s from %s", message.Id(), conn.Address())
	traceId := message.Id()
	context.Ctx.AsyncTaskPool().Schedule(func() {
		d.metering.TraceMessagePerformance(traceId)
		d.dispatcher.Dispatch(message, conn)
		d.metering.Stop(traceId)
	})
}
