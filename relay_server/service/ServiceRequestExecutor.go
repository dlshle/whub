package service

import (
	"wsdk/relay_common/connection"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/service"
	"wsdk/relay_server/context"
)

type InternalServiceRequestExecutor struct {
	handler service.IServiceHandler
}

func NewInternalServiceRequestExecutor(handler service.IServiceHandler) service.IRequestExecutor {
	return &InternalServiceRequestExecutor{handler}
}

func (e *InternalServiceRequestExecutor) Execute(request *service.ServiceRequest) {
	err := e.handler.Handle(request)
	if err != nil {
		request.Resolve(messages.NewErrorMessage(request.Id(), context.Ctx.Server().Id(), request.From(), request.Uri(), err.Error()))
	}
}

type RelayServiceRequestExecutor struct {
	conn   connection.IConnection
	hostId string
}

func NewRelayServiceRequestExecutor(c connection.IConnection) service.IRequestExecutor {
	e := &RelayServiceRequestExecutor{
		conn: c,
	}
	e.hostId = context.Ctx.Server().Id()
	return e
}

func (e *RelayServiceRequestExecutor) Execute(request *service.ServiceRequest) {
	response, err := e.conn.Request(request.Message)
	if request.Status() == service.ServiceRequestStatusDead {
		// last check on if messages is killed
		request.Resolve(messages.NewErrorMessage(request.Id(), e.hostId, request.From(), request.Uri(), "request has been cancelled or target server is dead"))
	} else if err != nil {
		request.Resolve(messages.NewErrorMessage(request.Id(), e.hostId, request.From(), request.Uri(), err.Error()))
	} else {
		request.Resolve(response)
	}
}
