package metering

import (
	"wsdk/relay_common/metering"
	"wsdk/relay_server/container"
	"wsdk/relay_server/context"
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
	return &ServerMeteringController{
		metering.NewMeteringController(context.Ctx.Logger().WithPrefix("[ServerMeteringController]")),
	}
}

func (c *ServerMeteringController) TraceMessagePerformance(messageId string) metering.IStopWatch {
	return c.Measure(c.GetAssembledTraceId(TMessagePerformance, messageId))
}

func init() {
	container.Container.Singleton(func() IServerMeteringController {
		return NewServerMeteringController()
	})
}
