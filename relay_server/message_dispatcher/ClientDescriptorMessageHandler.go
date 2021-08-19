package message_dispatcher

import (
	"errors"
	"fmt"
	base_conn "wsdk/common/connection"
	"wsdk/common/logger"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/message_actions"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/roles"
	"wsdk/relay_server/client"
	"wsdk/relay_server/container"
	"wsdk/relay_server/context"
	"wsdk/relay_server/core/client_manager"
	"wsdk/relay_server/core/connection_manager"
)

// TODO move this logic to client_management_service???
// Can't do this in cms as it doesn't have the connection info

// TODO new: Move Client Info Update logic to ClientManagementService!!

type ClientDescriptorMessageHandler struct {
	clientManager     client_manager.IClientManager         `$inject:""`
	connectionManager connection_manager.IConnectionManager `$inject:""`
	logger            *logger.SimpleLogger
}

func NewClientDescriptorMessageHandler() message_actions.IMessageHandler {
	handler := &ClientDescriptorMessageHandler{
		logger: context.Ctx.Logger().WithPrefix("[ClientDescriptorMessageHandler]"),
	}
	err := container.Container.Fill(handler)
	if err != nil {
		panic(err)
	}
	return handler
}

func (h *ClientDescriptorMessageHandler) Type() int {
	return messages.MessageTypeClientDescriptor
}

func (h *ClientDescriptorMessageHandler) Handle(message messages.IMessage, conn connection.IConnection) (err error) {
	if !base_conn.IsAsyncType(conn.ConnectionType()) {
		err = errors.New("non async connection type can not be used to initiate async client registration")
		conn.Send(messages.NewErrorResponseMessage(message, context.Ctx.Server().Id(), err.Error()))
		return err
	}
	if message == nil {
		err = errors.New("nil message")
		conn.Send(messages.NewErrorResponseMessage(message, context.Ctx.Server().Id(), err.Error()))
		return err
	}
	roleDescriptor, extraInfoDescriptor, err := client_manager.UnmarshallClientDescriptor(message)
	if err != nil {
		h.logger.Printf("failed to unmarshall descriptors from message by %s due to %s", conn.Address(), err.Error())
		conn.Send(messages.NewErrorResponseMessage(message, context.Ctx.Server().Id(), err.Error()))
		return err
	}
	if message.From() != roleDescriptor.Id {
		errMsg := fmt.Sprintf("client identity mismatch from(%s), descriptor(%s)", message.From(), roleDescriptor.Id)
		h.logger.Printf(errMsg)
		err = errors.New(errMsg)
		conn.Send(messages.NewErrorResponseMessage(message, context.Ctx.Server().Id(), errMsg))
		return err
	}

	client, err := h.clientManager.GetClient(message.From())
	if err != nil {
		errMsg := fmt.Sprintf("err while finding client %s from connection %s: %s", message.From(), conn.Address(), err.Error())
		h.logger.Println(errMsg)
		err = errors.New(errMsg)
		conn.Send(messages.NewErrorResponseMessage(message, context.Ctx.Server().Id(), err.Error()))
		return err
	}
	if client == nil {
		// client registration, do we do it here or route the request to the service?
		client, err = h.handleClientRegistration(roleDescriptor, extraInfoDescriptor)
		if err != nil {
			h.logger.Printf("err while conn %s registering client %s due to %s", conn.Address(), roleDescriptor.Id, err.Error())
			conn.Send(messages.NewErrorResponseMessage(message, context.Ctx.Server().Id(), err.Error()))
			return err
		}
		h.logger.Printf("conn %s has successfully registered client %s", conn.Address(), client.Id())
	}
	// login => associate connection with client info
	h.logger.Printf("handle client async connection(%s) login to %s", conn.Address(), client.Id())
	if err = h.handleClientLogin(client, extraInfoDescriptor.CKey, conn); err != nil {
		h.logger.Printf("conn %s unable to login client %s due to %s", conn.Address(), client.Id(), err.Error())
		conn.Send(messages.NewErrorResponseMessage(message, context.Ctx.Server().Id(), err.Error()))
		return err
	}
	h.logger.Printf("client async connection(%s) login to %s succeeded", conn.Address(), client.Id())
	return conn.Send(h.assembleServiceDescriptorMessageFrom(message))
}

func (h *ClientDescriptorMessageHandler) handleClientLogin(client *client.Client, cKey string, conn connection.IConnection) error {
	if client.CKey() != cKey {
		err := errors.New("invalid authorization(mismatched cKey)")
		return err
	}
	return h.connectionManager.RegisterClientToConnection(client.Id(), conn.Address())
}

func (h *ClientDescriptorMessageHandler) handleClientRegistration(desc roles.RoleDescriptor, extra roles.ClientExtraInfoDescriptor) (*client.Client, error) {
	client := client.NewClientFromDescriptor(desc, extra)
	err := h.clientManager.AddClient(client)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (h *ClientDescriptorMessageHandler) assembleServiceDescriptorMessageFrom(message messages.IMessage) messages.IMessage {
	return messages.NewMessage(message.Id(),
		context.Ctx.Server().Id(),
		message.From(),
		message.Uri(),
		messages.MessageTypeServerDescriptor,
		([]byte)(context.Ctx.Server().Describe().String()))
}
