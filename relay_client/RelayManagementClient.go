package relay_client

import (
	"errors"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/service"
)

const (
	ServerServiceManagerUri = service.ServicePrefix + "/relay"
)

// ServerServiceManager
const (
	ServiceManagerRegisterService   = ServerServiceManagerUri + "/register"   // payload = service descriptor
	ServiceManagerUnregisterService = ServerServiceManagerUri + "/unregister" // payload = service descriptor
	ServiceManagerUpdateService     = ServerServiceManagerUri + "/update"     // payload = service descriptor
)

type IServiceManagerClient interface {
	RegisterService(descriptor service.ServiceDescriptor) error
	UnregisterService(descriptor service.ServiceDescriptor) error
	UpdateService(descriptor service.ServiceDescriptor) error
	Response(message *messages.Message) error
	HealthCheck() error
}

type ServiceManagerClient struct {
	clientId   string
	serverId   string
	serverConn connection.IConnection
}

func NewServiceCenterClient(id string, serverId string, conn connection.IConnection) IServiceManagerClient {
	return &ServiceManagerClient{
		clientId:   id,
		serverId:   serverId,
		serverConn: conn,
	}
}

func (c *ServiceManagerClient) draftDescriptorMessageWith(uri string, descriptor service.ServiceDescriptor) *messages.Message {
	return c.draftMessage(
		uri,
		messages.MessageTypeServiceRequest,
		([]byte)(descriptor.String()),
	)
}

func (c *ServiceManagerClient) requestMessage(message *messages.Message) (err error) {
	resp, err := c.serverConn.Request(message)
	if resp != nil && resp.MessageType() == messages.MessageTypeError {
		return errors.New((string)(resp.Payload()))
	}
	return
}

func (c *ServiceManagerClient) HealthCheck() error {
	return c.requestMessage(c.draftMessage("", messages.MessageTypePing, nil))
}

func (c *ServiceManagerClient) RegisterService(descriptor service.ServiceDescriptor) (err error) {
	return c.requestMessage(c.draftDescriptorMessageWith(ServiceManagerRegisterService, descriptor))
}

func (c *ServiceManagerClient) UnregisterService(descriptor service.ServiceDescriptor) (err error) {
	return c.requestMessage(c.draftDescriptorMessageWith(ServiceManagerUnregisterService, descriptor))
}

func (c *ServiceManagerClient) UpdateService(descriptor service.ServiceDescriptor) error {
	return c.requestMessage(c.draftDescriptorMessageWith(ServiceManagerUpdateService, descriptor))
}

func (c *ServiceManagerClient) Response(message *messages.Message) error {
	return c.serverConn.Send(message)
}

func (c *ServiceManagerClient) draftMessage(uri string, msgType int, payload []byte) *messages.Message {
	return messages.DraftMessage(c.clientId, c.serverId, uri, msgType, payload)
}
