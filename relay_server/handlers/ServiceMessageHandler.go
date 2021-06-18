package handlers

import (
	"errors"
	"strings"
	"wsdk/relay_common"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/service"
	"wsdk/relay_server"
	service2 "wsdk/relay_server/service"
)

// ServiceRequestHandler
type ServiceRequestHandler struct {
	ctx            *relay_common.WRContext
	serviceManager service2.IServiceManager
}

func (h *ServiceRequestHandler) NewServiceRequestHandler(ctx *relay_common.WRContext, manager service2.IServiceManager) relay_server.IServerMessageHandler {
	return &ServiceRequestHandler{ctx, manager}
}

func (h *ServiceRequestHandler) Handle(message *messages.Message, next messages.NextMessageHandler) (*messages.Message, error) {
	if !strings.HasPrefix(message.Uri(), service.ServicePrefix) {
		return next(message)
		// return nil, errors.New(relay_server.NewInvalidServiceRequestUriError(message.Uri()).Json())
	}
	service := h.serviceManager.MatchServiceByUri(message.Uri())
	if service == nil {
		return nil, errors.New(service2.NewCanNotFindServiceError(message.Uri()).Json())
	}
	return service.Request(message), nil
}

func (h *ServiceRequestHandler) Priority() int {
	return relay_server.HandlerPriorityServiceRequest
}
