package relay_client

import (
	"wsdk/relay_client/context"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/service"
)

type ClientServiceExecutor struct {
	handler service.IServiceHandler
}

func NewClientServiceExecutor(handler service.IServiceHandler) *ClientServiceExecutor {
	return &ClientServiceExecutor{
		handler: handler,
	}
}

func (e *ClientServiceExecutor) Execute(request *service.ServiceRequest) {
	err := e.handler.Handle(request)
	if err != nil {
		request.Resolve(messages.NewErrorMessage(request.Id(), context.Ctx.Identity().Id(), request.From(), request.Uri(), err.Error()))
	}
}
