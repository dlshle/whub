package message_actions

import (
	"wsdk/relay_common/connection"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/roles"
)

type IMessageHandler interface {
	Type() int
	Types() []int
	Handle(messages.IMessage, connection.IConnection) error
}

// common message handlers

type PingMessageHandler struct {
	role roles.IDescribableRole
}

func NewPingMessageHandler(role roles.IDescribableRole) IMessageHandler {
	return &PingMessageHandler{role}
}

func (h *PingMessageHandler) Handle(message messages.IMessage, conn connection.IConnection) error {
	var resp messages.IMessage
	if message.To() != h.role.Id() {
		resp = messages.NewInternalErrorMessage(message.Id(), h.role.Id(), message.From(), message.Uri(), "incorrect receiver id")
	} else {
		resp = messages.NewPongMessage(message.Id(), h.role.Id(), message.From())
	}
	return conn.Send(resp)
}

func (h *PingMessageHandler) Type() int {
	return messages.MessageTypePing
}

func (h *PingMessageHandler) Types() []int {
	return nil
}

type InvalidMessageHandler struct {
	role roles.IDescribableRole
}

func NewInvalidMessageHandler(role roles.IDescribableRole) IMessageHandler {
	return &InvalidMessageHandler{role}
}

func (h *InvalidMessageHandler) Handle(message messages.IMessage, conn connection.IConnection) error {
	return conn.Send(messages.NewInternalErrorMessage(message.Id(), h.role.Id(), message.From(), message.Uri(), "invalid message(no handler found)"))
}

func (h *InvalidMessageHandler) Type() int {
	return messages.MessageTypeUnknown
}

func (h *InvalidMessageHandler) Types() []int {
	return nil
}
