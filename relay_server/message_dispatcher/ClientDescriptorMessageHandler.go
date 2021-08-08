package message_dispatcher

import (
	"encoding/json"
	"errors"
	"fmt"
	base_conn "wsdk/common/connection"
	"wsdk/common/logger"
	"wsdk/common/utils"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/message_actions"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/roles"
	"wsdk/relay_server/client"
	"wsdk/relay_server/container"
	"wsdk/relay_server/context"
	"wsdk/relay_server/controllers/anonymous_client_manager"
	"wsdk/relay_server/controllers/client_manager"
	"wsdk/relay_server/controllers/connection_manager"
)

type ClientDescriptorMessageHandler struct {
	anonymousClientManager anonymous_client_manager.IAnonymousClientManager `$inject:""`
	clientManager          client_manager.IClientManager                    `$inject:""`
	connectionManager      connection_manager.IConnectionManager            `$inject:""`
	logger                 *logger.SimpleLogger
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

func (h *ClientDescriptorMessageHandler) Handle(message *messages.Message, conn connection.IConnection) (err error) {
	if !base_conn.IsAsyncType(conn.ConnectionType()) {
		return errors.New("non async connection type can not be used to initiate async client registration")
	}
	if message == nil {
		return errors.New("nil message")
	}
	roleDescriptor, extraInfoDescriptor, err := h.unmarshallClientDescriptor(message)
	if err != nil {
		h.logger.Printf("failed to unmarshall descriptors from message by %s due to %s", conn.Address(), err.Error())
		conn.Send(messages.NewErrorResponseMessage(message, context.Ctx.Server().Id(), err.Error()))
		return err
	}
	// TODO remove later
	anonymousClient := h.anonymousClientManager.GetClient(conn.Address())
	if c, err := h.connectionManager.GetConnectionByAddress(conn.Address()); c != nil && err == nil && anonymousClient != nil {
		h.logger.Printf("handle anonymous client(%s) promotion", anonymousClient.Address())
		// registration
		h.handleClientRegistration(roleDescriptor, extraInfoDescriptor, conn)
		serverDescMsg := messages.NewMessage(message.Id(),
			context.Ctx.Server().Id(),
			message.From(),
			message.Uri(),
			messages.MessageTypeServerDescriptor,
			([]byte)(context.Ctx.Server().Describe().String()))
		return conn.Send(serverDescMsg)
	}
	// client_manager info update
	client := h.clientManager.GetClient(message.From())
	if client == nil {
		h.logger.Printf("can not find the client by %s from connection %s.", message.From(), conn.Address())
		err = errors.New(fmt.Sprintf("can not find client by id %s, conn %s", message.From(), conn.Address()))
		conn.Send(messages.NewErrorResponseMessage(message, context.Ctx.Server().Id(), err.Error()))
		return err
	}
	h.logger.Println("update client with ", roleDescriptor, extraInfoDescriptor)
	client.SetDescription(roleDescriptor.Description)
	client.SetCKey(extraInfoDescriptor.CKey)
	client.SetPScope(extraInfoDescriptor.PScope)
	return conn.Send(messages.NewACKMessage(message.Id(), context.Ctx.Server().Id(), message.From(), message.Uri()))
}

func (h *ClientDescriptorMessageHandler) unmarshallClientDescriptor(message *messages.Message) (roleDescriptor roles.RoleDescriptor, extraInfoDescriptor roles.ClientExtraInfoDescriptor, err error) {
	err = utils.ProcessWithError([]func() error{
		func() error {
			return json.Unmarshal(message.Payload(), &roleDescriptor)
		},
		func() error {
			return json.Unmarshal(([]byte)(roleDescriptor.ExtraInfo), &extraInfoDescriptor)
		},
	})
	return
}

func (h *ClientDescriptorMessageHandler) handleClientRegistration(clientDescriptor roles.RoleDescriptor, clientExtraInfo roles.ClientExtraInfoDescriptor, conn connection.IConnection) {
	// TODO remove this later
	h.anonymousClientManager.RemoveClient(conn.Address())
	client := client.NewClient(conn, clientDescriptor.Id, clientDescriptor.Description, clientExtraInfo.CType, clientExtraInfo.CKey, clientExtraInfo.PScope)
	h.connectionManager.RegisterClientToConnection(client.Id(), conn.Address())
	h.clientManager.AcceptClient(client.Id(), client)
}
