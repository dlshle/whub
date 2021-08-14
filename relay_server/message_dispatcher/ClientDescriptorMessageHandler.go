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
	"wsdk/relay_server/controllers/client_manager"
	"wsdk/relay_server/controllers/connection_manager"
)

// TODO move this logic to client_management_service???
// Can't do this in cms as it doesn't have the connection info

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
	if c, err := h.connectionManager.GetConnectionByAddress(conn.Address()); c != nil && err == nil {
		h.logger.Printf("handle client connection(%s) promotion", conn.Address())
		// registration
		h.handleClientConnectionRegistration(roleDescriptor, extraInfoDescriptor, conn)
		serverDescMsg := messages.NewMessage(message.Id(),
			context.Ctx.Server().Id(),
			message.From(),
			message.Uri(),
			messages.MessageTypeServerDescriptor,
			([]byte)(context.Ctx.Server().Describe().String()))
		return conn.Send(serverDescMsg)
	}
	// client_manager info update
	client, err := h.clientManager.GetClient(message.From())
	if err != nil {
		errMsg := fmt.Sprintf("err while finding client %s from connection %s: %s", message.From(), conn.Address(), err.Error())
		h.logger.Println(errMsg)
		err = errors.New(errMsg)
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

func (h *ClientDescriptorMessageHandler) handleClientConnectionRegistration(clientDescriptor roles.RoleDescriptor, clientExtraInfo roles.ClientExtraInfoDescriptor, conn connection.IConnection) {
	client := client.NewClient(clientDescriptor.Id, clientDescriptor.Description, clientExtraInfo.CType, clientExtraInfo.CKey, clientExtraInfo.PScope)
	h.connectionManager.RegisterClientToConnection(client.Id(), conn.Address())
}
