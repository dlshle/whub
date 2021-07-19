package message_dispatcher

import (
	"encoding/json"
	"errors"
	"fmt"
	"wsdk/common/logger"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/message_actions"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/roles"
	"wsdk/relay_common/utils"
	"wsdk/relay_server/client"
	"wsdk/relay_server/container"
	"wsdk/relay_server/context"
	"wsdk/relay_server/controllers"
)

type ClientDescriptorMessageHandler struct {
	anonymousClientManager controllers.IAnonymousClientManager `$inject:""`
	clientManager          controllers.IClientManager          `$inject:""`
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
	if message == nil {
		return errors.New("nil message")
	}
	roleDescriptor, extraInfoDescriptor, err := h.unmarshallClientDescriptor(message)
	if err != nil {
		h.logger.Printf("failed to unmarshall descriptors from message by %s due to %s", conn.Address(), err.Error())
		conn.Send(messages.NewErrorResponseMessage(message, context.Ctx.Server().Id(), err.Error()))
		return err
	}
	anonymousClient := h.anonymousClientManager.GetClient(conn.Address())
	if anonymousClient != nil {
		h.logger.Printf("handle anonymous client(%s) promotion", anonymousClient.Address())
		// promote
		h.handleClientPromotion(roleDescriptor, extraInfoDescriptor, anonymousClient)
		return conn.Send(messages.NewACKMessage(message.Id(), context.Ctx.Server().Id(), message.From(), message.Uri()))
	}
	// client info update
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

func (h *ClientDescriptorMessageHandler) handleClientPromotion(clientDescriptor roles.RoleDescriptor, clientExtraInfo roles.ClientExtraInfoDescriptor, anonymousClient *client.Client) {
	h.anonymousClientManager.RemoveClient(anonymousClient.Id())
	client := client.NewClient(anonymousClient.Connection(), clientDescriptor.Id, clientDescriptor.Description, clientExtraInfo.CType, clientExtraInfo.CKey, clientExtraInfo.PScope)
	h.clientManager.AcceptClient(client.Id(), client)
}
