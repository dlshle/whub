package service

import (
	"errors"
	"wsdk/relay_common"
	"wsdk/relay_common/messages"
)

const (
	ServerServiceManagerUri = ServicePrefix + "/relay"
)

// ServerServiceManager
const (
	ServiceManagerRegisterService   = ServerServiceManagerUri + "/register"   // payload = service descriptor
	ServiceManagerUnregisterService = ServerServiceManagerUri + "/unregister" // payload = service descriptor
	ServiceManagerUpdateService     = ServerServiceManagerUri + "/update"     // payload = service descriptor
)

type IServiceManagerClient interface {
	RegisterService(descriptor ServiceDescriptor) error
	UnregisterService(descriptor ServiceDescriptor) error
	UpdateService(descriptor ServiceDescriptor) error
	Response(message *messages.Message) error
	HealthCheck() error
}

type ServiceManagerClient struct {
	clientCtx *relay_common.WRContext
	server    *relay_common.WRServer
}

func NewServiceCenterClient(ctx *relay_common.WRContext, server *relay_common.WRServer) IServiceManagerClient {
	return &ServiceManagerClient{
		clientCtx: ctx,
		server:    server,
	}
}

func (c *ServiceManagerClient) draftDescriptorMessageWith(uri string, descriptor ServiceDescriptor) *messages.Message {
	return c.draftMessage(
		uri,
		messages.MessageTypeClientNotification,
		([]byte)(descriptor.String()),
	)
}

func (c *ServiceManagerClient) requestMessage(message *messages.Message) (err error) {
	resp, err := c.server.Request(message)
	if resp != nil && resp.MessageType() == messages.MessageTypeError {
		return errors.New((string)(resp.Payload()))
	}
	return
}

func (c *ServiceManagerClient) HealthCheck() error {
	return c.requestMessage(c.draftMessage("", messages.MessageTypePing, nil))
}

func (c *ServiceManagerClient) RegisterService(descriptor ServiceDescriptor) (err error) {
	return c.requestMessage(c.draftDescriptorMessageWith(ServiceManagerRegisterService, descriptor))
}

func (c *ServiceManagerClient) UnregisterService(descriptor ServiceDescriptor) (err error) {
	return c.requestMessage(c.draftDescriptorMessageWith(ServiceManagerUnregisterService, descriptor))
}

func (c *ServiceManagerClient) UpdateService(descriptor ServiceDescriptor) error {
	return c.requestMessage(c.draftDescriptorMessageWith(ServiceManagerUpdateService, descriptor))
}

func (c *ServiceManagerClient) Response(message *messages.Message) error {
	return c.server.Send(message)
}

func (c *ServiceManagerClient) draftMessage(uri string, msgType int, payload []byte) *messages.Message {
	return c.server.DraftMessage(c.clientCtx.Identity().Id(), c.server.Id(), uri, msgType, payload)
}
