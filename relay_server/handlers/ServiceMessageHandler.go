package handlers

import (
	"errors"
	"strings"
	"wsdk/relay_common"
	"wsdk/relay_common/messages"
	"wsdk/relay_server"
)

// ServiceMessageHandler
type ServiceMessageHandler struct {
	ctx *relay_common.WRContext
	serviceManager relay_server.IServiceManager
}

func (h *ServiceMessageHandler) NewServiceMessageHandler(ctx *relay_common.WRContext, manager relay_server.IServiceManager) relay_server.IServerMessageHandler {
	return &ServiceMessageHandler{ctx, manager}
}

func (h *ServiceMessageHandler) Handle(message *messages.Message, next messages.NextMessageHandler) (*messages.Message, error) {
	if !strings.HasPrefix(message.Uri(), relay_common.ServicePrefix) {
		return next(message)
		// return nil, errors.New(relay_server.NewInvalidServiceMessageUriError(message.Uri()).Json())
	}
	service := h.serviceManager.MatchServiceByUri(message.Uri())
	if service == nil {
		return nil, errors.New(relay_server.NewCanNotFindServiceError(message.Uri()).Json())
	}
	return service.Request(message), nil
}

func (h *ServiceMessageHandler) Priority() int {
	return relay_server.HandlerPriorityServiceMessage
}
