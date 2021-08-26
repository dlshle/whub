package request

import (
	"errors"
	"fmt"
	"wsdk/common/logger"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/service"
	"wsdk/relay_server/container"
	"wsdk/relay_server/context"
	"wsdk/relay_server/core/connection_manager"
	server_errors "wsdk/relay_server/errors"
	"wsdk/relay_server/events"
)

type InternalServiceRequestExecutor struct {
	handler service.IServiceHandler
}

func NewInternalServiceRequestExecutor(handler service.IServiceHandler) service.IRequestExecutor {
	return &InternalServiceRequestExecutor{handler}
}

func (e *InternalServiceRequestExecutor) Execute(request service.IServiceRequest) {
	// internal service will resolve the request if no error is present
	err := e.handler.Handle(request)
	if err != nil {
		request.Resolve(messages.NewInternalErrorMessage(request.Id(), context.Ctx.Server().Id(), request.From(), request.Uri(), server_errors.NewJsonMessageError(err.Error())))
	}
}

type RelayServiceRequestExecutor struct {
	serviceId         string
	providerId        string
	hostId            string
	connections       []connection.IConnection
	lastSucceededConn int
	connectionManager connection_manager.IConnectionManager `$inject:""`
	logger            *logger.SimpleLogger
}

func NewRelayServiceRequestExecutor(serviceId string, providerId string) service.IRequestExecutor {
	e := &RelayServiceRequestExecutor{
		hostId:     context.Ctx.Server().Id(),
		serviceId:  serviceId,
		providerId: providerId,
		logger:     context.Ctx.Logger().WithPrefix(fmt.Sprintf("[RelayServiceRequestExecutor-%s]", serviceId)),
	}
	err := container.Container.Fill(e)
	if err != nil {
		panic(err)
	}
	err = e.updateConnections()
	if err != nil {
		e.connections = []connection.IConnection{}
	}
	e.initNotifications()
	return e
}

func (e *RelayServiceRequestExecutor) initNotifications() {
	// do not on ClientConnectionEstablished event because new client connection doesn't mean the client is ready for
	// service requests
	events.OnEvent(events.EventServiceNewProvider, e.handleNewServiceProviderEvent)
	events.OnEvent(events.EventClientConnectionClosed, e.handleClientConnectionChangeEvent)
	events.OnEvent(events.EventClientConnectionGone, e.handleClientConnectionChangeEvent)
}

func (e *RelayServiceRequestExecutor) handleNewServiceProviderEvent(event messages.IMessage) {
	if (string)(event.Payload()) == e.serviceId {
		e.updateConnections()
	}
}

func (e *RelayServiceRequestExecutor) handleClientConnectionChangeEvent(event messages.IMessage) {
	if (string)(event.Payload()) == e.providerId {
		e.updateConnections()
	}
}

func (e *RelayServiceRequestExecutor) updateConnections() error {
	conns, err := e.connectionManager.GetConnectionsByClientId(e.providerId)
	if err != nil {
		e.logger.Printf("update connection failed due to %s", err.Error())
		return err
	}
	e.connections = conns
	e.logger.Println("connections:", conns)
	return nil
}

func (e *RelayServiceRequestExecutor) Execute(request service.IServiceRequest) {
	response, err := e.doRequest(request)
	if request.Status() == service.ServiceRequestStatusDead {
		// last check on if message_dispatcher is killed
		request.Resolve(messages.NewInternalErrorMessage(request.Id(), e.hostId, request.From(), request.Uri(), "request has been cancelled or target server is dead"))
	} else if err != nil {
		request.Resolve(messages.NewInternalErrorMessage(request.Id(), e.hostId, request.From(), request.Uri(), err.Error()))
	} else {
		request.Resolve(response)
	}
}

// try all connections from lastSucceededConn until one succeeded
func (e *RelayServiceRequestExecutor) doRequest(request service.IServiceRequest) (msg messages.IMessage, err error) {
	if len(e.connections) == 0 {
		return nil, errors.New("all service connection is down")
	}
	size := len(e.connections)
	for i := 0; i < size; i++ {
		e.lastSucceededConn++
		if msg, err = e.connections[(e.lastSucceededConn % len(e.connections))].Request(request.Message()); err == nil {
			// TODO remove later
			e.logger.Printf("conn #%d go the request", e.lastSucceededConn%len(e.connections))
			// once the first connection successfully handles the request, return
			return
		}
	}
	return
}
