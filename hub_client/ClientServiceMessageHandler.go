package hub_client

import (
	"fmt"
	"whub/hub_client/container"
	"whub/hub_client/context"
	"whub/hub_client/controllers"
	"whub/hub_common/connection"
	"whub/hub_common/messages"
	"whub/hub_common/middleware"
	"whub/hub_common/service"
)

type ClientServiceMessageHandler struct {
	m       controllers.IClientMeteringController `$inject:""`
	manager IServiceManager                       `$inject:""`
}

func NewClientServiceMessageHandler() *ClientServiceMessageHandler {
	h := &ClientServiceMessageHandler{}
	err := container.Container.Fill(h)
	if err != nil {
		panic(err)
	}
	return h
}

func (h *ClientServiceMessageHandler) Type() int {
	return messages.MessageTypeServiceRequest
}

func (h *ClientServiceMessageHandler) Types() []int {
	return service.ServiceRequestMessageHandlerTypes
}

func (h *ClientServiceMessageHandler) Handle(msg messages.IMessage, conn connection.IConnection) error {
	h.m.Track(h.m.GetAssembledTraceId(controllers.TMessagePerformance, msg.Id()), "in service handler")
	matchContext, err := h.manager.MatchServiceByUri(msg.Uri())
	if err != nil || matchContext.Value == nil {
		h.m.Stop(h.m.GetAssembledTraceId(controllers.TMessagePerformance, msg.Id()))
		return conn.Send(messages.NewInternalErrorMessage(
			msg.Id(),
			context.Ctx.Identity().Id(),
			msg.From(),
			msg.Uri(),
			fmt.Sprintf(`{"error":"%s"}`, err.Error()),
			msg.Headers(),
		))
	}
	svc := matchContext.Value.(IClientService)
	request := service.NewServiceRequest(msg)
	request.SetContext("uri_pattern", matchContext.UriPattern)
	request.SetContext("path_params", matchContext.PathParams)
	request.SetContext("query_params", matchContext.QueryParams)
	// at least run the common middleware
	request = middleware.ConnectionTypeMiddleware(conn, request)
	resp := svc.Handle(request)

	// request die here
	request.Free()
	h.m.Stop(h.m.GetAssembledTraceId(controllers.TMessagePerformance, msg.Id()))
	return conn.Send(resp)
}
