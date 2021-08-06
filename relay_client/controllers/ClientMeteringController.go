package controllers

import (
	"wsdk/relay_client/container"
	"wsdk/relay_client/context"
	"wsdk/relay_common/metering"
)

const (
	TMessagePerformance = "TMessagePerformance"
)

type ClientMeteringController struct {
	metering.IMeteringController
}

type IClientMeteringController interface {
	metering.IMeteringController
	TraceMessagePerformance(messageId string) metering.IStopWatch
}

func NewClientMeteringController() IClientMeteringController {
	return &ClientMeteringController{
		metering.NewMeteringController(context.Ctx.Logger().WithPrefix("[ServerMeteringController]")),
	}
}

func (c *ClientMeteringController) TraceMessagePerformance(messageId string) metering.IStopWatch {
	return c.Measure(c.GetAssembledTraceId(TMessagePerformance, messageId))
}

func init() {
	container.Container.Singleton(func() IClientMeteringController {
		return NewClientMeteringController()
	})
}
