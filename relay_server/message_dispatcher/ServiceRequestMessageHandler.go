package message_dispatcher

import (
	base_conn "wsdk/common/connection"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/message_actions"
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

var serviceRequestMessageHandlerTypes []int

func init() {
	serviceRequestMessageHandlerTypes = []int{
		messages.MessageTypeServiceRequest,
		messages.MessageTypeServiceGetRequest,
		messages.MessageTypeServicePostRequest,
		messages.MessageTypeServicePutRequest,
		messages.MessageTypeServicePatchRequest,
		messages.MessageTypeServiceDeleteRequest,
		messages.MessageTypeServiceOptionsRequest,
		messages.MessageTypeServiceHeadRequest,
	}
}

type ServiceRequestMessageHandler struct {
	serviceManager    service_manager.IServiceManager       `$inject:""`
	middlewareManager middleware_manager.IMiddlewareManager `$inject:""`
	metering          metering.IServerMeteringController    `$inject:""`
}

func NewServiceRequestMessageHandler() message_actions.IMessageHandler {
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
	return serviceRequestMessageHandlerTypes
}

func (h *ServiceRequestMessageHandler) Handle(message messages.IMessage, conn connection.IConnection) (err error) {
	h.metering.Track(h.metering.GetAssembledTraceId(metering.TMessagePerformance, message.Id()), "in service handler")
	// remove redundant / at the end of the uri
	uri := message.Uri()
	if len(uri) > 2 && uri[len(uri)-1] == '/' && uri[len(uri)-2] != '/' {
		message = message.SetUri(uri[:len(uri)-1])
	}
	svc := h.serviceManager.FindServiceByUri(message.Uri())
	if svc == nil {
		err = service_base.NewCanNotFindServiceError(message.Uri())
		conn.Send(messages.NewErrorMessage(message.Id(), context.Ctx.Server().Id(), message.From(), message.Uri(),
			messages.MessageTypeSvcNotFoundError,
			errors.NewJsonMessageError(err.Error())))
		h.metering.Stop(h.metering.GetAssembledTraceId(metering.TMessagePerformance, message.Id()))
		return err
	}
	request := h.middlewareManager.RunMiddlewares(conn, service.NewServiceRequest(message))
	response := svc.Handle(request)
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
