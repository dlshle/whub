package metering

import (
	"wsdk/relay_common/messages"
	"wsdk/relay_common/metering"
	"wsdk/relay_server/container"
	"wsdk/relay_server/context"
	"wsdk/relay_server/events"
	"wsdk/relay_server/module_base"
)

const (
	TMessagePerformance = "TMessagePerformance"
)

type MeteringModule struct {
	*module_base.ModuleBase
	metering.IMeteringController
}

type IMeteringModule interface {
	metering.IMeteringController
	TraceMessagePerformance(messageId string) metering.IStopWatch
}

func NewMeteringModule() IMeteringModule {
	controller := &MeteringModule{
		IMeteringController: metering.NewMeteringController(context.Ctx.Logger().WithPrefix("[MeteringModule]")),
	}
	controller.initNotifications()
	return controller
}

func (m *MeteringModule) Init() error {
	m.ModuleBase = module_base.NewModuleBase("Metering", func() error {
		var holder IMeteringModule
		m.disposeNotifications()
		return container.Container.RemoveByType(holder)
	})
	m.IMeteringController = metering.NewMeteringController(m.Logger())
	return container.Container.Singleton(func() IMeteringModule {
		return m
	})
}

func (c *MeteringModule) handleServerClose(message messages.IMessage) {
	c.StopAll()
}

func (c *MeteringModule) initNotifications() {
	events.OnEvent(events.EventServerClosed, c.handleServerClose)
}

func (c *MeteringModule) disposeNotifications() {
	events.OffEvent(events.EventServerClosed, c.handleServerClose)
}

func (c *MeteringModule) TraceMessagePerformance(messageId string) metering.IStopWatch {
	return c.Measure(c.GetAssembledTraceId(TMessagePerformance, messageId))
}

func Load() error {
	return container.Container.Singleton(func() IMeteringModule {
		return NewMeteringModule()
	})
}
