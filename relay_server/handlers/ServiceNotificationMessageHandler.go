package handlers

import (
	"encoding/json"
	"wsdk/relay_common"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/messages"
	common "wsdk/relay_common/service"
	"wsdk/relay_common/utils"
	"wsdk/relay_server/service"
)

type ServiceNotificationMessageHandler struct {
	ctx *relay_common.WRContext
	m   *service.ServiceManager
}

func (h *ServiceNotificationMessageHandler) Type() int {
	return messages.MessageTypeClientServiceNotification
}

func (h *ServiceNotificationMessageHandler) Handle(message *messages.Message, conn *connection.WRConnection) (err error) {
	var serviceDescriptor common.ServiceDescriptor
	return utils.ProcessWithErrors(func() error {
		return json.Unmarshal(message.Payload(), &serviceDescriptor)
	}, func() error {
		return h.m.UpdateService(serviceDescriptor)
	}, func() error {
		return conn.Send(messages.NewErrorMessage(message.Id(), h.ctx.Identity().Id(), message.From(), message.Uri(), err.Error()))
	})
}
