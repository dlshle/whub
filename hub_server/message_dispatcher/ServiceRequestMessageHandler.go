package message_dispatcher

import (
	"fmt"
	base_conn "whub/common/connection"
	"whub/common/uri_trie"
	"whub/hub_common/connection"
	"whub/hub_common/dispatcher"
	"whub/hub_common/messages"
	"whub/hub_common/service"
	"whub/hub_server/context"
	"whub/hub_server/errors"
	"whub/hub_server/module_base"
	"whub/hub_server/modules/metering"
	"whub/hub_server/modules/middleware_manager"
	"whub/hub_server/modules/service_manager"
	"whub/hub_server/service_base"
)

type ServiceRequestMessageHandler struct {
	serviceManager    service_manager.IServiceManagerModule       `module:""`
	middlewareManager middleware_manager.IMiddlewareManagerModule `module:""`
	metering          metering.IMeteringModule                    `module:""`
}

func NewServiceRequestMessageHandler() dispatcher.IMessageHandler {
	handler := &ServiceRequestMessageHandler{}
	err := module_base.Manager.AutoFill(handler)
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
		h.metering.Stop(h.metering.GetAssembledTraceId(metering.TMessagePerformance, message.Id()))
		// service panic handling
		if recovered := recover(); recovered != nil {
			conn.Send(messages.NewErrorResponse(message, context.Ctx.Server().Id(),
				messages.MessageTypeSvcInternalError,
				errors.NewJsonMessageError(fmt.Sprintf("unknown server internal error occurred: %v", recovered))))
		}
	}()
	h.metering.Track(h.metering.GetAssembledTraceId(metering.TMessagePerformance, message.Id()), "in service handler")
	message = h.processIncomingMessage(message)
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
	request := h.createRequest(message, matchContext, conn)

	var response messages.IMessage
	if request.Status() > service.ServiceRequestStatusProcessing {
		// request is resolved in middleware
		response = request.Response()
	} else {
		// continue the request with service
		response = svc.Handle(request)
	}
	// request die here
	request.Free()

	if response == nil && !base_conn.IsAsyncType(conn.ConnectionType()) {
		err = conn.Send(messages.NewErrorResponse(request, context.Ctx.Server().Id(),
			messages.MessageTypeSvcForbiddenError,
			errors.NewJsonMessageError("service does not support sync requests")))
	} else {
		err = conn.Send(response)
	}
	return
}

func (h *ServiceRequestMessageHandler) processIncomingMessage(message messages.IMessage) messages.IMessage {
	// remove redundant / at the end of the uri
	uri := message.Uri()
	if len(uri) > 2 && uri[len(uri)-1] == '/' && uri[len(uri)-2] != '/' {
		message = message.SetUri(uri[:len(uri)-1])
	}
	return message
}

func (h *ServiceRequestMessageHandler) createRequest(message messages.IMessage, matchContext *uri_trie.MatchContext, conn connection.IConnection) service.IServiceRequest {
	request := service.NewServiceRequest(message)
	request = h.registerRequestMetaContext(request, matchContext)
	return h.middlewareManager.RunMiddlewares(conn, request)
}

func (h *ServiceRequestMessageHandler) registerRequestMetaContext(request service.IServiceRequest, matchContext *uri_trie.MatchContext) service.IServiceRequest {
	request.SetContext(service.ServiceRequestContextUriPattern, matchContext.UriPattern)
	request.SetContext(service.ServiceRequestContextPathParams, matchContext.PathParams)
	request.SetContext(service.ServiceRequestContextQueryParams, matchContext.QueryParams)
	return request
}
