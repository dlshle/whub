package handlers

import (
"wsdk/relay_common"
"wsdk/relay_common/messages"
"wsdk/relay_server"
)

// ClientMessageHandler
type ClientMessageHandler struct {
	ctx *relay_common.WRContext
	clientManager relay_server.IClientManager
}

func (h *ClientMessageHandler) NewClientMessageHandler(ctx *relay_common.WRContext, manager relay_server.IClientManager) relay_server.IServerMessageHandler {
	return &ClientMessageHandler{ctx, manager}
}

func (h *ClientMessageHandler) Handle(message *messages.Message, next messages.NextMessageHandler) (*messages.Message, error) {
	client := h.clientManager.GetClient(message.To())
	if client == nil {
		return next(message)
	}
	return client.Request(message)
}

func (h *ClientMessageHandler) Priority() int {
	return relay_server.HandlerPriorityClientMessage
}
