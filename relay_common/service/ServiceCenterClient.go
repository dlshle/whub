package service

import (
	"errors"
	"wsdk/relay_common"
	"wsdk/relay_common/messages"
)

type IServiceCenterClient interface {
	RegisterService(descriptor ServiceDescriptor) error
	UnregisterService(descriptor ServiceDescriptor) error
	UpdateService(descriptor ServiceDescriptor) error
	Response(message *messages.Message) error
	HealthCheck() error
}

type ServiceCenterClient struct {
	clientCtx *relay_common.WRContext
	server *relay_common.WRServer
}
func NewServiceCenterClient(ctx *relay_common.WRContext, server *relay_common.WRServer) IServiceCenterClient {
	return &ServiceCenterClient{
		clientCtx: ctx,
		server: server,
	}
}

func (c *ServiceCenterClient) draftDescriptorMessageWith(uri string, descriptor ServiceDescriptor) *messages.Message {
	return c.draftMessage(
		uri,
		messages.MessageTypeClientNotification, ([]byte)(descriptor.String()),
	)
}

func (c *ServiceCenterClient) requestMessage(message *messages.Message) (err error) {
	resp, err := c.server.Request(message)
	if resp != nil && resp.MessageType() == messages.MessageTypeError {
		return errors.New((string)(resp.Payload()))
	}
	return
}

func (c *ServiceCenterClient) HealthCheck() error {
	return c.requestMessage(c.draftMessage("", messages.MessageTypePing, nil))
}

func (c *ServiceCenterClient) RegisterService(descriptor ServiceDescriptor) (err error) {
	return c.requestMessage(c.draftDescriptorMessageWith(ServiceCenterRegisterService, descriptor))
}

func (c *ServiceCenterClient) UnregisterService(descriptor ServiceDescriptor) (err error) {
	return c.requestMessage(c.draftDescriptorMessageWith(ServiceCenterUnregisterService, descriptor))
}

func (c *ServiceCenterClient) UpdateService(descriptor ServiceDescriptor) error {
	return c.requestMessage(c.draftDescriptorMessageWith(ServiceCenterUpdateService, descriptor))
}

func (c *ServiceCenterClient) Response(message *messages.Message) error {
	return c.server.Send(message)
}

func (c *ServiceCenterClient) draftMessage(uri string, msgType int, payload []byte) *messages.Message {
	return c.server.DraftMessage(c.clientCtx.Identity().Id(), c.server.Id(), uri, msgType, payload)
}
