package relay_client

import (
	"fmt"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/service"
)

type ClientServiceExecutor struct {
	ctx     IContext
	handler service.IServiceHandler
}

func NewClientServiceExecutor(ctx IContext, handler service.IServiceHandler) *ClientServiceExecutor {
	return &ClientServiceExecutor{
		ctx:     ctx,
		handler: handler,
	}
}

func (e *ClientServiceExecutor) Execute(request *service.ServiceRequest) {
	if !e.handler.SupportsUri(request.Uri()) {
		request.Resolve(messages.NewErrorMessage(request.Id(), request.To(), e.ctx.Identity().Id(), request.Uri(), fmt.Sprintf("unsupported uri_trie %s", request.Uri())))
		return
	}
	e.handler.Handle(request)
}
