package relay_client

import (
	"fmt"
	"wsdk/relay_client/container"
	"wsdk/relay_client/context"
	"wsdk/relay_client/controllers"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/messages"
	"wsdk/relay_server/core/metering"
)

type ClientServiceMessageHandler struct {
	m       controllers.IClientMeteringController `$inject:""`
	service IClientService
}

func NewClientServiceMessageHandler() *ClientServiceMessageHandler {
	h := &ClientServiceMessageHandler{}
	err := container.Container.Fill(h)
	if err != nil {
		panic(err)
	}
	return h
}

func (h *ClientServiceMessageHandler) SetService(svc IClientService) {
	h.service = svc
}

func (h *ClientServiceMessageHandler) Type() int {
	return messages.MessageTypeServiceRequest
}

func (h *ClientServiceMessageHandler) Handle(msg *messages.Message, conn connection.IConnection) error {
	h.m.Track(h.m.GetAssembledTraceId(metering.TMessagePerformance, msg.Id()), "in service handler")
	if h.service == nil {
		h.m.Stop(h.m.GetAssembledTraceId(metering.TMessagePerformance, msg.Id()))
		return conn.Send(messages.NewErrorMessage(
			msg.Id(),
			context.Ctx.Identity().Id(),
			msg.From(),
			msg.Uri(),
			fmt.Sprintf("client connection %s does not have service running yet", context.Ctx.Identity().Id()),
		))
	}
	if !h.service.SupportsUri(msg.Uri()) {
		h.m.Stop(h.m.GetAssembledTraceId(metering.TMessagePerformance, msg.Id()))
		return conn.Send(messages.NewErrorMessage(msg.Id(),
			context.Ctx.Identity().Id(),
			msg.From(),
			msg.Uri(),
			fmt.Sprintf("uri %s is not supported by service %s", msg.Uri(), h.service.Id())))
	}
	resp := h.service.Handle(msg)
	err := conn.Send(resp)
	h.m.Stop(h.m.GetAssembledTraceId(metering.TMessagePerformance, msg.Id()))
	return err
}
