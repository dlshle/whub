package message_dispatcher

import (
	"wsdk/relay_common/connection"
	"wsdk/relay_common/message_actions"
	"wsdk/relay_common/messages"
	"wsdk/relay_server/container"
	"wsdk/relay_server/context"
	service "wsdk/relay_server/controllers/service_manager"
	"wsdk/relay_server/service_base"
)

type ServiceRequestMessageHandler struct {
	manager service.IServiceManager `$inject:""`
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
	svc := h.manager.FindServiceByUri(message.Uri())
	if svc == nil {
		err = service_base.NewCanNotFindServiceError(message.Uri())
		conn.Send(messages.NewErrorMessage(message.Id(), context.Ctx.Server().Id(), message.From(), message.Uri(), err.Error()))
		return err
	}
	response := svc.Handle(message)
	return conn.Send(response)
}
