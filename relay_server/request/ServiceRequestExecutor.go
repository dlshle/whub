package request

import (
	"wsdk/relay_common/connection"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/service"
	"wsdk/relay_server/context"
)

type InternalServiceRequestExecutor struct {
	ctx     *context.Context
	handler service.IServiceHandler
}

func NewInternalServiceRequestExecutor(ctx *context.Context, handler service.IServiceHandler) service.IRequestExecutor {
	return &InternalServiceRequestExecutor{ctx, handler}
}

func (e *InternalServiceRequestExecutor) Execute(request *service.ServiceRequest) {
	err := e.handler.Handle(request)
	if err != nil {
		request.Resolve(messages.NewErrorMessage(request.Id(), e.ctx.Server().Id(), request.From(), request.Uri(), err.Error()))
	}
}

type RelayServiceRequestExecutor struct {
	ctx  *context.Context
	conn *connection.Connection
}

func RelayRequestExecutor(ctx *context.Context, c *connection.Connection) service.IRequestExecutor {
	return &RelayServiceRequestExecutor{
		ctx:  ctx,
		conn: c,
	}
}

func (e *RelayServiceRequestExecutor) Execute(request *service.ServiceRequest) {
	// check if messages is processable
	if service.UnProcessableServiceRequestMap[request.Status()] {
		request.Resolve(messages.NewErrorMessage(request.Id(), e.ctx.Server().Id(), request.From(), request.Uri(), "request has been cancelled or target server is dead"))
		return
	}
	response, err := e.conn.Request(request.Message)
	if request.Status() == service.ServiceRequestStatusDead {
		// last check on if messages is killed
		request.Resolve(messages.NewErrorMessage(request.Id(), e.ctx.Server().Id(), request.From(), request.Uri(), "request has been cancelled or target server is dead"))
	} else if err != nil {
		request.Resolve(messages.NewErrorMessage(request.Id(), e.ctx.Server().Id(), request.From(), request.Uri(), err.Error()))
	} else {
		request.Resolve(response)
	}
}
