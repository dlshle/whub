package clients

import (
	"errors"
	"wsdk/relay_client/connections"
	"wsdk/relay_client/container"
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

type IRelayServiceClient interface {
	RegisterService(descriptor service.ServiceDescriptor) error
	UnregisterService(descriptor service.ServiceDescriptor) error
	UpdateService(descriptor service.ServiceDescriptor) error
	Response(message messages.IMessage) error
	HealthCheck() error
}

type RelayServiceClient struct {
	clientId   string
	serverConn connection.IConnection
	connPool   connections.IConnectionPool `$inject:""`
}

func NewRelayServiceClient(id string, conn connection.IConnection) IRelayServiceClient {
	client := &RelayServiceClient{
		clientId:   id,
		serverConn: conn,
	}
	err := container.Container.Fill(client)
	if err != nil {
		panic(err)
	}
	container.Container.Singleton(func() IRelayServiceClient {
		return client
	})
	return client
}

func (c *RelayServiceClient) draftDescriptorMessageWith(uri string, descriptor service.ServiceDescriptor) messages.IMessage {
	return c.draftMessage(
		uri,
		messages.MessageTypeServiceRequest,
		([]byte)(descriptor.String()),
	)
}

func (c *RelayServiceClient) requestMessage(message messages.IMessage) (err error) {
	resp, err := c.serverConn.Request(message)
	if resp != nil && resp.IsErrorMessage() {
		return errors.New((string)(resp.Payload()))
	}
	return
}

func (c *RelayServiceClient) HealthCheck() error {
	return c.requestMessage(c.draftMessage("", messages.MessageTypePing, nil))
}

func (c *RelayServiceClient) RegisterService(descriptor service.ServiceDescriptor) (err error) {
	return c.requestMessage(c.draftDescriptorMessageWith(ServiceManagerRegisterService, descriptor))
}

func (c *RelayServiceClient) UnregisterService(descriptor service.ServiceDescriptor) (err error) {
	return c.requestMessage(c.draftDescriptorMessageWith(ServiceManagerUnregisterService, descriptor))
}

func (c *RelayServiceClient) UpdateService(descriptor service.ServiceDescriptor) error {
	return c.requestMessage(c.draftDescriptorMessageWith(ServiceManagerUpdateService, descriptor))
}

func (c *RelayServiceClient) Response(message messages.IMessage) error {
	return c.serverConn.Send(message)
}

func (c *RelayServiceClient) draftMessage(uri string, msgType int, payload []byte) messages.IMessage {
	return messages.DraftMessage(c.clientId, "", uri, msgType, payload)
}
