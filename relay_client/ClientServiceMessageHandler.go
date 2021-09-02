package relay_client

import (
	"wsdk/relay_client/container"
	"wsdk/relay_client/context"
	"wsdk/relay_client/controllers"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/service"
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
			err.Error(),
		))
	}
	svc := matchContext.Value.(IClientService)
	request := service.NewServiceRequest(msg)
	request.SetContext("uri_pattern", matchContext.UriPattern)
	request.SetContext("path_params", matchContext.PathParams)
	request.SetContext("query_params", matchContext.QueryParams)
	resp := svc.Handle(request)
	h.m.Stop(h.m.GetAssembledTraceId(controllers.TMessagePerformance, msg.Id()))
	return conn.Send(resp)
}
