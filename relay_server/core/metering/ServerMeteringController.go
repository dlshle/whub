package metering

import (
	"wsdk/relay_common/messages"
	"wsdk/relay_common/metering"
	"wsdk/relay_server/container"
	"wsdk/relay_server/context"
	"wsdk/relay_server/events"
)

const (
	TMessagePerformance = "TMessagePerformance"
)

type ServerMeteringController struct {
	metering.IMeteringController
}

type IServerMeteringController interface {
	metering.IMeteringController
	TraceMessagePerformance(messageId string) metering.IStopWatch
}

func NewServerMeteringController() IServerMeteringController {
	controller := &ServerMeteringController{
		metering.NewMeteringController(context.Ctx.Logger().WithPrefix("[ServerMeteringController]")),
	}
	controller.initNotifications()
	return controller
}

func (c *ServerMeteringController) initNotifications() {
	events.OnEvent(events.EventServerClosed, func(message messages.IMessage) {
		c.StopAll()
	})
}

func (c *ServerMeteringController) TraceMessagePerformance(messageId string) metering.IStopWatch {
	return c.Measure(c.GetAssembledTraceId(TMessagePerformance, messageId))
}

func init() {
	container.Container.Singleton(func() IServerMeteringController {
		return NewServerMeteringController()
	})
}
