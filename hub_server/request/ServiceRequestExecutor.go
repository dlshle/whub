package request

import (
	"errors"
	"fmt"
	"whub/common/logger"
	"whub/hub_common/connection"
	"whub/hub_common/messages"
	"whub/hub_common/service"
	"whub/hub_server/context"
	server_errors "whub/hub_server/errors"
	"whub/hub_server/events"
	"whub/hub_server/module_base"
	"whub/hub_server/modules/connection_manager"
)

type InternalServiceRequestExecutor struct {
	handler service.IDefaultServiceHandler
}

func NewInternalServiceRequestExecutor(handler service.IDefaultServiceHandler) service.IRequestExecutor {
	return &InternalServiceRequestExecutor{handler}
}

func (e *InternalServiceRequestExecutor) Execute(request service.IServiceRequest) {
	// internal service will resolve the request if no error is present
	err := e.handler.Handle(request)
	if err != nil {
		request.Resolve(messages.NewInternalErrorMessage(request.Id(), context.Ctx.Server().Id(), request.From(), request.Uri(), server_errors.NewJsonMessageError(err.Error()), request.Headers()))
	}
}

type RelayServiceRequestExecutor struct {
	serviceId         string
	providerId        string
	hostId            string
	connections       []connection.IConnection
	lastSucceededConn int
	connectionManager connection_manager.IConnectionManagerModule `module:""`
	logger            *logger.SimpleLogger
}

func NewRelayServiceRequestExecutor(serviceId string, providerId string) *RelayServiceRequestExecutor {
	e := &RelayServiceRequestExecutor{
		hostId:     context.Ctx.Server().Id(),
		serviceId:  serviceId,
		providerId: providerId,
		logger:     context.Ctx.Logger().WithPrefix(fmt.Sprintf("[RelayServiceRequestExecutor-%s]", serviceId)),
	}
	err := module_base.Manager.AutoFill(e)
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
	// events.OnEvent(events.EventServiceNewProvider, e.handleNewServiceProviderEvent)
	events.OnEvent(events.EventClientConnectionClosed, e.handleClientConnectionChangeEvent)
	events.OnEvent(events.EventClientConnectionGone, e.handleClientConnectionChangeEvent)
}

func (e *RelayServiceRequestExecutor) handleClientConnectionChangeEvent(event messages.IMessage) {
	if (string)(event.Payload()) == e.providerId {
		e.updateConnections()
	}
}

func (e *RelayServiceRequestExecutor) updateConnections() error {
	for i := 0; i < len(e.connections); i++ {
		conn := e.connections[i]
		if conn == nil || !conn.IsLive() {
			// remove this conn
			e.connections = append(e.connections[:i], e.connections[i+1:]...)
			i--
		}
	}
	e.logger.Println("service connections:", e.connections)
	return nil
}

func (e *RelayServiceRequestExecutor) Execute(request service.IServiceRequest) {
	response, err := e.doRequest(request)
	if request.Status() == service.ServiceRequestStatusDead {
		// last check on if message_dispatcher is killed
		request.Resolve(messages.NewInternalErrorMessage(request.Id(), e.hostId, request.From(), request.Uri(), `{"error":"request has been cancelled or target server is dead"}"`, request.Headers()))
	} else if err != nil {
		request.Resolve(messages.NewInternalErrorMessage(request.Id(), e.hostId, request.From(), request.Uri(), fmt.Sprintf(`{"error":"%s"}`,err.Error()), request.Headers()))
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
			// once the first connection successfully handles the request, return
			return
		}
	}
	return
}

func (e *RelayServiceRequestExecutor) UpdateProviderConnection(connAddr string) (err error) {
	e.logger.Printf("update provider connection: %s", connAddr)
	for _, c := range e.connections {
		if c.Address() == connAddr {
			err = errors.New(fmt.Sprintf("connection address %s has already been added to the executor", connAddr))
			e.logger.Printf(err.Error())
			return err
		}
	}
	conn, err := e.connectionManager.GetConnectionByAddress(connAddr)
	if err != nil {
		err = errors.New(fmt.Sprintf("connection address %s has already been added to the executor", connAddr))
		e.logger.Printf(err.Error())
		return err
	}
	e.connections = append(e.connections, conn)
	e.logger.Printf("update provider connection %s succeeded", connAddr)
	return nil
}

func (e *RelayServiceRequestExecutor) GetProviderConnections() []connection.IConnection {
	return e.connections
}
