package message_dispatcher

import (
	"fmt"
	base_conn "wsdk/common/connection"
	"wsdk/common/uri_trie"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/dispatcher"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/service"
	"wsdk/relay_server/container"
	"wsdk/relay_server/context"
	"wsdk/relay_server/core/metering"
	"wsdk/relay_server/core/middleware_manager"
	"wsdk/relay_server/core/service_manager"
	"wsdk/relay_server/errors"
	"wsdk/relay_server/service_base"
)

type ServiceRequestMessageHandler struct {
	serviceManager    service_manager.IServiceManager       `$inject:""`
	middlewareManager middleware_manager.IMiddlewareManager `$inject:""`
	metering          metering.IServerMeteringController    `$inject:""`
}

func NewServiceRequestMessageHandler() dispatcher.IMessageHandler {
	handler := &ServiceRequestMessageHandler{}
	err := container.Container.Fill(handler)
	if err != nil {
		panic(err)
	}
	return handler
}

func (h *ServiceRequestMessageHandler) Type() int {
	return messages.MessageTypeServiceRequest
}

func (h *ServiceRequestMessageHandler) Types() []int {
	return service.ServiceRequestMessageHandlerTypes
}

func (h *ServiceRequestMessageHandler) Handle(message messages.IMessage, conn connection.IConnection) (err error) {
	defer func() {
		// service panic handling
		if recovered := recover(); recovered != nil {
			conn.Send(messages.NewErrorResponse(message, context.Ctx.Server().Id(),
				messages.MessageTypeSvcInternalError,
				errors.NewJsonMessageError(fmt.Sprintf("unknown server internal error occurred: %v", recovered))))
			h.metering.Stop(h.metering.GetAssembledTraceId(metering.TMessagePerformance, message.Id()))
		}
	}()
	h.metering.Track(h.metering.GetAssembledTraceId(metering.TMessagePerformance, message.Id()), "in service handler")
	// remove redundant / at the end of the uri
	uri := message.Uri()
	if len(uri) > 2 && uri[len(uri)-1] == '/' && uri[len(uri)-2] != '/' {
		message = message.SetUri(uri[:len(uri)-1])
	}
	matchContext := h.serviceManager.MatchServiceByUri(message.Uri())
	if matchContext == nil {
		err = service_base.NewCanNotFindServiceError(message.Uri())
		conn.Send(messages.NewErrorMessage(message.Id(), context.Ctx.Server().Id(), message.From(), message.Uri(),
			messages.MessageTypeSvcNotFoundError,
			errors.NewJsonMessageError(err.Error())))
		h.metering.Stop(h.metering.GetAssembledTraceId(metering.TMessagePerformance, message.Id()))
		return err
	}
	svc := matchContext.Value.(service_base.IService)
	request := service.NewServiceRequest(message)
	request = h.registerRequestMetaContext(request, matchContext)
	request = h.middlewareManager.RunMiddlewares(conn, request)
	// free match context
	matchContext = nil

	var response messages.IMessage
	if request.Status() > service.ServiceRequestStatusProcessing {
		// request is resolved in middleware
		response = request.Response()
	} else {
		// continue the request with service
		response = svc.Handle(request)
	}

	if response == nil && !base_conn.IsAsyncType(conn.ConnectionType()) {
		err = conn.Send(messages.NewErrorResponse(request, context.Ctx.Server().Id(),
			messages.MessageTypeSvcForbiddenError,
			errors.NewJsonMessageError("service does not support sync requests")))
	} else {
		err = conn.Send(response)
	}
	h.metering.Stop(h.metering.GetAssembledTraceId(metering.TMessagePerformance, message.Id()))
	return
}

func (h *ServiceRequestMessageHandler) registerRequestMetaContext(request service.IServiceRequest, matchContext *uri_trie.MatchContext) service.IServiceRequest {
	request.SetContext(service.ServiceRequestContextUriPattern, matchContext.UriPattern)
	request.SetContext(service.ServiceRequestContextPathParams, matchContext.PathParams)
	request.SetContext(service.ServiceRequestContextQueryParams, matchContext.QueryParams)
	return request
}
