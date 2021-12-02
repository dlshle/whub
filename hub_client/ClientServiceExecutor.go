package hub_client

import (
	"whub/hub_client/context"
	"whub/hub_common/messages"
	"whub/hub_common/service"
)

type ClientServiceExecutor struct {
	handler service.IDefaultServiceHandler
}

func NewClientServiceExecutor(handler service.IDefaultServiceHandler) *ClientServiceExecutor {
	return &ClientServiceExecutor{
		handler: handler,
	}
}

func (e *ClientServiceExecutor) Execute(request service.IServiceRequest) {
	err := e.handler.Handle(request)
	if err != nil {
		request.Resolve(messages.NewInternalErrorMessage(request.Id(), context.Ctx.Identity().Id(), request.From(), request.Uri(), err.Error()))
	}
}
