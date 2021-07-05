package relay_server

import (
	"encoding/json"
	common_connection "wsdk/relay_common/connection"
	"wsdk/relay_common/message_actions"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/roles"
	"wsdk/relay_common/utils"
	"wsdk/relay_server/client"
	"wsdk/relay_server/container"
	"wsdk/relay_server/context"
	"wsdk/relay_server/managers"
	"wsdk/websocket/connection"
)

type clientExtraInfoDescriptor struct {
	PScope int    `json:"pScope"`
	CKey   string `json:"cKey"`
	CType  int    `json:"cType"`
}

type ClientConnectionHandler struct {
	messageDispatcher      message_actions.IMessageDispatcher
	clientManager          managers.IClientManager
	anonymousClientManager managers.IAnonymousClientManager
}

type IClientConnectionHandler interface {
	HandleConnectionEstablished(conn *connection.WsConnection)
}

func NewClientConnectionHandler(messageDispatcher message_actions.IMessageDispatcher) IClientConnectionHandler {
	return &ClientConnectionHandler{
		messageDispatcher:      messageDispatcher,
		clientManager:          container.Container.GetById(managers.ClientManagerId).(managers.IClientManager),
		anonymousClientManager: container.Container.GetById(managers.AnonymousClientManagerId).(managers.IAnonymousClientManager),
	}
}

func (h *ClientConnectionHandler) HandleConnectionEstablished(conn *connection.WsConnection) {
	rawConn := common_connection.NewConnection(conn, common_connection.DefaultTimeout, context.Ctx.MessageParser(), context.Ctx.NotificationEmitter())
	// any message from any connection needs to go through here
	rawConn.OnIncomingMessage(func(message *messages.Message) {
		h.messageDispatcher.Dispatch(message, rawConn)
	})
	rawClient := client.NewAnonymousClient(rawConn)
	h.anonymousClientManager.AcceptClient(rawClient.Address(), rawClient)
	resp, err := rawClient.Request(rawClient.NewMessage(context.Ctx.Server().Id(), "", messages.MessageTypeServerDescriptor, ([]byte)(context.Ctx.Server().Describe().String())))
	// try to handle anonymous client upgrade
	if err == nil && resp.MessageType() == messages.MessageTypeClientDescriptor {
		h.handleClientPromotionMessage(rawClient, resp)
	}
}

func (h *ClientConnectionHandler) handleClientPromotionMessage(anonymousClient *client.Client, message *messages.Message) {
	var clientDescriptor roles.RoleDescriptor
	var clientExtraInfo clientExtraInfoDescriptor
	err := utils.ProcessWithError([]func() error{
		func() error {
			return json.Unmarshal(message.Payload(), &clientDescriptor)
		},
		func() error {
			return json.Unmarshal(([]byte)(clientDescriptor.ExtraInfo), &clientExtraInfo)
		},
	})
	if err == nil {
		// client promotion
		h.anonymousClientManager.RemoveClient(anonymousClient.Id())
		client := client.NewClient(anonymousClient.Connection(), clientDescriptor.Id, clientDescriptor.Description, clientExtraInfo.CType, clientExtraInfo.CKey, clientExtraInfo.PScope)
		h.clientManager.AcceptClient(client.Id(), client)
	}
}
