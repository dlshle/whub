package metering

import (
	"wsdk/relay_common/messages"
	"wsdk/relay_common/metering"
	"wsdk/relay_server/events"
	"wsdk/relay_server/module_base"
)

const (
	ID                  = "Metering"
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

func (m *MeteringModule) Init() error {
	m.ModuleBase = module_base.NewModuleBase(ID, func() error {
		m.disposeNotifications()
		return nil
	})
	m.IMeteringController = metering.NewMeteringController(m.Logger())
	return nil
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
