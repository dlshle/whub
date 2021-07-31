package relay_client

import (
	"fmt"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/messages"
)

type ClientServiceMessageHandler struct {
	service IClientService
}

func NewClientServiceMessageHandler() *ClientServiceMessageHandler {
	return &ClientServiceMessageHandler{}
}

func (s *ClientServiceMessageHandler) SetService(svc IClientService) {
	s.service = svc
}

func (s *ClientServiceMessageHandler) Type() int {
	return messages.MessageTypeServiceRequest
}

func (s *ClientServiceMessageHandler) Handle(msg *messages.Message, conn connection.IConnection) error {
	if s.service == nil {
		return conn.Send(messages.NewErrorMessage(
			msg.Id(),
			Ctx.Identity().Id(),
			msg.From(),
			msg.Uri(),
			fmt.Sprintf("client connection %s does not have service running yet", Ctx.Identity().Id()),
		))
	}
	if !s.service.SupportsUri(msg.Uri()) {
		return conn.Send(messages.NewErrorMessage(msg.Id(),
			Ctx.Identity().Id(),
			msg.From(),
			msg.Uri(),
			fmt.Sprintf("uri %s is not supported by service %s", msg.Uri(), s.service.Id())))
	}
	return conn.Send(s.service.Handle(msg))
}
