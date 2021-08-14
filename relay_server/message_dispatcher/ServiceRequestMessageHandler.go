package message_dispatcher

import (
	"wsdk/relay_common/connection"
	"wsdk/relay_common/message_actions"
	"wsdk/relay_common/messages"
	"wsdk/relay_server/container"
	"wsdk/relay_server/context"
	"wsdk/relay_server/core/metering"
	service "wsdk/relay_server/core/service_manager"
	"wsdk/relay_server/service_base"
)

type ServiceRequestMessageHandler struct {
	manager  service.IServiceManager            `$inject:""`
	metering metering.IServerMeteringController `$inject:""`
}

func NewServiceRequestMessageHandler() message_actions.IMessageHandler {
	handler := &ServiceRequestMessageHandler{}
	err := container.Container.Fill(handler)
	if err != nil {
		panic(err)
	}
	return handler
}

func (h *ServiceRequestMessageHandler) Type() int {
	return messages.MessageTypeServiceRequest
}

func (h *ServiceRequestMessageHandler) Handle(message *messages.Message, conn connection.IConnection) (err error) {
	h.metering.Track(h.metering.GetAssembledTraceId(metering.TMessagePerformance, message.Id()), "in service handler")
	// remove redundant / at the end of the uri
	uri := message.Uri()
	if uri[len(uri)-1] == '/' {
		message = message.SetUri(uri[:len(uri)-1])
	}
	svc := h.manager.FindServiceByUri(message.Uri())
	if svc == nil {
		err = service_base.NewCanNotFindServiceError(message.Uri())
		conn.Send(messages.NewErrorMessage(message.Id(), context.Ctx.Server().Id(), message.From(), message.Uri(), err.Error()))
		h.metering.Stop(h.metering.GetAssembledTraceId(metering.TMessagePerformance, message.Id()))
		return err
	}
	response := svc.Handle(message)
	if response == nil {
		err = conn.Send(messages.NewErrorMessage(message.Id(), context.Ctx.Server().Id(), message.From(), message.Uri(), "service does not support sync requests"))
	} else {
		err = conn.Send(response)
	}
	h.metering.Stop(h.metering.GetAssembledTraceId(metering.TMessagePerformance, message.Id()))
	return
}
