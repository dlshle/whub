package handlers

import (
	"wsdk/relay_common/connection"
	"wsdk/relay_common/message_actions"
	"wsdk/relay_common/messages"
	"wsdk/relay_server/container"
	"wsdk/relay_server/context"
	"wsdk/relay_server/managers"
	"wsdk/relay_server/service"
)

type ServiceRequestMessageHandler struct {
	manager managers.IServiceManager
}

func NewServiceRequestMessageHandler() message_actions.IMessageHandler {
	return &ServiceRequestMessageHandler{
		manager: container.Container.GetById(managers.ServiceManagerId).(managers.IServiceManager),
	}
}

func (h *ServiceRequestMessageHandler) Type() int {
	return messages.MessageTypeServiceRequest
}

func (h *ServiceRequestMessageHandler) Handle(message *messages.Message, conn connection.IConnection) (err error) {
	svc := h.manager.FindServiceByUri(message.Uri())
	if svc == nil {
		err = service.NewCanNotFindServiceError(message.Uri())
		conn.Send(messages.NewErrorMessage(message.Id(), context.Ctx.Server().Id(), message.From(), message.Uri(), err.Error()))
		return err
	}
	return conn.Send(svc.Handle(message))
}
