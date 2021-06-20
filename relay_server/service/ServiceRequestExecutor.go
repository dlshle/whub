package service

import (
	"wsdk/relay_common"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/service"
)

type ServerServiceRequestExecutor struct {
	ctx  *relay_common.WRContext
	conn *connection.WRConnection
}

func SimpleMessageRequestExecutor(ctx *relay_common.WRContext, c *connection.WRConnection) *ServerServiceRequestExecutor {
	return &ServerServiceRequestExecutor{
		ctx:  ctx,
		conn: c,
	}
}

func (e *ServerServiceRequestExecutor) Execute(request *service.ServiceRequest) {
	// check if messages is processable
	if service.UnProcessableServiceRequestMap[request.Status()] {
		request.Resolve(messages.NewErrorMessage(request.Id(), request.From(), e.ctx.Identity().Id(), request.Uri(), "request has been cancelled or target server is dead"))
		return
	}
	request.TransitStatus(service.ServiceRequestStatusProcessing)
	response, err := e.conn.Request(request.Message)
	if request.Status() == service.ServiceRequestStatusDead {
		// last check on if messages is killed
		request.Resolve(messages.NewErrorMessage(request.Id(), request.From(), e.ctx.Identity().Id(), request.Uri(), "request has been cancelled or target server is dead"))
	} else if err != nil {
		request.Resolve(messages.NewErrorMessage(request.Id(), request.From(), e.ctx.Identity().Id(), request.Uri(), err.Error()))
	} else {
		request.Resolve(response)
	}
}
