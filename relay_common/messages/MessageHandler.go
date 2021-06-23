package messages

import (
	"wsdk/relay_common"
	"wsdk/relay_common/connection"
)

type IMessageHandler interface {
	Type() int
	Handle(*Message, *connection.WRConnection) error
}

// common message handlers

type PingMessageHandler struct {
	role relay_common.IDescribableRole
}

func NewPingMessageHandler(role relay_common.IDescribableRole) IMessageHandler {
	return &PingMessageHandler{role}
}

func (h *PingMessageHandler) Handle(message *Message, conn *connection.WRConnection) error {
	var resp *Message
	if message.to != h.role.Id() {
		resp = NewErrorMessage(message.Id(), h.role.Id(), message.From(), message.Uri(), "incorrect receiver id")
	} else {
		resp = NewPongMessage(message.Id(), h.role.Id(), message.From())
	}
	return conn.Send(resp)
}

func (h *PingMessageHandler) Type() int {
	return MessageTypePing
}

type InvalidMessageHandler struct {
	role relay_common.IDescribableRole
}

func NewInvalidMessageHandler(role relay_common.IDescribableRole) IMessageHandler {
	return &InvalidMessageHandler{role}
}

func (h *InvalidMessageHandler) Handle(message *Message, conn *connection.WRConnection) error {
	return conn.Send(NewErrorMessage(message.Id(), h.role.Id(), message.From(), message.Uri(), "invalid message(no handler found)"))
}

func (h *InvalidMessageHandler) Type() int {
	return MessageTypeUnknown
}
