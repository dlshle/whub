package service

import (
	"wsdk/relay_common"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/service"
)

type InternalServiceRequestExecutor struct {
	ctx     *relay_common.WRContext
	handler service.IServiceHandler
}

func NewInternalServiceRequestExecutor(ctx *relay_common.WRContext, handler service.IServiceHandler) relay_common.IRequestExecutor {
	return &InternalServiceRequestExecutor{ctx, handler}
}

func (e *InternalServiceRequestExecutor) Execute(request *service.ServiceRequest) {
	err := e.handler.Handle(request)
	if err != nil {
		request.Resolve(messages.NewErrorMessage(request.Id(), e.ctx.Identity().Id(), request.From(), request.Uri(), err.Error()))
	}
}

type RelayServiceRequestExecutor struct {
	ctx  *relay_common.WRContext
	conn *connection.WRConnection
}

func RelayRequestExecutor(ctx *relay_common.WRContext, c *connection.WRConnection) relay_common.IRequestExecutor {
	return &RelayServiceRequestExecutor{
		ctx:  ctx,
		conn: c,
	}
}

func (e *RelayServiceRequestExecutor) Execute(request *service.ServiceRequest) {
	// check if messages is processable
	if service.UnProcessableServiceRequestMap[request.Status()] {
		request.Resolve(messages.NewErrorMessage(request.Id(), e.ctx.Identity().Id(), request.From(), request.Uri(), "request has been cancelled or target server is dead"))
		return
	}
	request.TransitStatus(service.ServiceRequestStatusProcessing)
	response, err := e.conn.Request(request.Message)
	if request.Status() == service.ServiceRequestStatusDead {
		// last check on if messages is killed
		request.Resolve(messages.NewErrorMessage(request.Id(), e.ctx.Identity().Id(), request.From(), request.Uri(), "request has been cancelled or target server is dead"))
	} else if err != nil {
		request.Resolve(messages.NewErrorMessage(request.Id(), e.ctx.Identity().Id(), request.From(), request.Uri(), err.Error()))
	} else {
		request.Resolve(response)
	}
}
