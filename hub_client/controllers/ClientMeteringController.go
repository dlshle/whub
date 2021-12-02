package controllers

import (
	"whub/hub_client/container"
	"whub/hub_client/context"
	"whub/hub_common/metering"
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
		metering.NewMeteringController(context.Ctx.Logger().WithPrefix("[MeteringModule]")),
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
