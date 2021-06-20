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
	ctx *relay_common.WRContext
}

func NewPingMessageHandler(ctx *relay_common.WRContext) IMessageHandler {
	return &PingMessageHandler{ctx}
}

func (h *PingMessageHandler) Handle(message *Message, conn *connection.WRConnection) error {
	var resp *Message
	if message.to != h.ctx.Identity().Id() {
		resp = NewErrorMessage(message.Id(), h.ctx.Identity().Id(), message.From(), message.Uri(), "incorrect receiver id")
	} else {
		resp = NewPongMessage(message.Id(), h.ctx.Identity().Id(), message.From())
	}
	return conn.Send(resp)
}

func (h *PingMessageHandler) Type() int {
	return MessageTypePing
}

type InvalidMessageHandler struct {
	ctx *relay_common.WRContext
}

func NewInvalidMessageHandler(ctx *relay_common.WRContext) IMessageHandler {
	return &InvalidMessageHandler{ctx}
}

func (h *InvalidMessageHandler) Handle(message *Message, conn *connection.WRConnection) error {
	return conn.Send(NewErrorMessage(message.Id(), h.ctx.Identity().Id(), message.From(), message.Uri(), "invalid message(no handler found)"))
}

func (h *InvalidMessageHandler) Type() int {
	return MessageTypeUnknown
}
