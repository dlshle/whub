package handlers

import (
	"encoding/json"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/message_actions"
	"wsdk/relay_common/messages"
	common "wsdk/relay_common/service"
	"wsdk/relay_common/utils"
	"wsdk/relay_server/context"
	"wsdk/relay_server/service"
)

type ServiceNotificationMessageHandler struct {
	ctx *context.Context
	m   *service.ServiceManager
}

func NewServiceNotificationMessageHandler(ctx *context.Context, manager *service.ServiceManager) message_actions.IMessageHandler {
	return &ServiceNotificationMessageHandler{ctx: ctx, m: manager}
}

func (h *ServiceNotificationMessageHandler) Type() int {
	return messages.MessageTypeClientServiceNotification
}

func (h *ServiceNotificationMessageHandler) Handle(message *messages.Message, conn *connection.Connection) (err error) {
	var serviceDescriptor common.ServiceDescriptor
	err = utils.ProcessWithErrors(func() error {
		return json.Unmarshal(message.Payload(), &serviceDescriptor)
	}, func() error {
		return h.m.UpdateService(serviceDescriptor)
	})
	if err != nil {
		err = conn.Send(messages.NewErrorMessage(message.Id(), h.ctx.Server().Id(), message.From(), message.Uri(), err.Error()))
	} else {
		err = conn.Send(messages.NewACKMessage(message.Id(), h.ctx.Server().Id(), message.From(), message.Uri()))
	}
	return err
}
