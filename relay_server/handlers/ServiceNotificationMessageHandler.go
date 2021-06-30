package handlers

import (
	"encoding/json"
	"fmt"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/message_actions"
	"wsdk/relay_common/messages"
	common "wsdk/relay_common/service"
	"wsdk/relay_common/utils"
	"wsdk/relay_server/container"
	"wsdk/relay_server/context"
	"wsdk/relay_server/managers"
	"wsdk/relay_server/service"
)

type ServiceNotificationMessageHandler struct {
	serviceManager managers.IServiceManager
}

func NewServiceNotificationMessageHandler() message_actions.IMessageHandler {
	return &ServiceNotificationMessageHandler{
		container.Container.GetById(managers.ServiceManagerId).(managers.IServiceManager),
	}
}

func (h *ServiceNotificationMessageHandler) Type() int {
	return messages.MessageTypeClientServiceNotification
}

func (h *ServiceNotificationMessageHandler) findAndHandleRequest(message *messages.Message, conn connection.IConnection) error {
	svc := h.serviceManager.FindServiceByUri(message.Uri())
	if svc == nil {
		return service.NewCanNotFindServiceError(message.Uri())
	}
	return conn.Send(svc.Handle(message))
}

func (h *ServiceNotificationMessageHandler) Handle(message *messages.Message, conn connection.IConnection) (err error) {
	var serviceDescriptor common.ServiceDescriptor
	err = utils.ProcessWithErrors(func() error {
		fmt.Println((string)(message.Payload()))
		return json.Unmarshal(message.Payload(), &serviceDescriptor)
	}, func() error {
		return h.serviceManager.UpdateService(serviceDescriptor)
	})
	if err != nil {
		// log error
		conn.Send(messages.NewErrorMessage(message.Id(), context.Ctx.Server().Id(), message.From(), message.Uri(), err.Error()))
		return err
	} else {
		return conn.Send(messages.NewACKMessage(message.Id(), context.Ctx.Server().Id(), message.From(), message.Uri()))
	}
}
