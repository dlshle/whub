package relay_server

import (
	"fmt"
	"wsdk/common/logger"
	common_connection "wsdk/relay_common/connection"
	"wsdk/relay_common/message_actions"
	"wsdk/relay_common/messages"
	"wsdk/relay_server/client"
	"wsdk/relay_server/container"
	"wsdk/relay_server/context"
	"wsdk/relay_server/managers"
	"wsdk/websocket/connection"
)

type ClientConnectionHandler struct {
	messageDispatcher      message_actions.IMessageDispatcher
	clientManager          managers.IClientManager
	anonymousClientManager managers.IAnonymousClientManager
	logger                 *logger.SimpleLogger
}

type IClientConnectionHandler interface {
	HandleConnectionEstablished(conn *connection.WsConnection)
}

func NewClientConnectionHandler(messageDispatcher message_actions.IMessageDispatcher) IClientConnectionHandler {
	h := &ClientConnectionHandler{
		messageDispatcher:      messageDispatcher,
		clientManager:          container.Container.GetById(managers.ClientManagerId).(managers.IClientManager),
		anonymousClientManager: container.Container.GetById(managers.AnonymousClientManagerId).(managers.IAnonymousClientManager),
		logger:                 context.Ctx.Logger().WithPrefix("[ClientConnectionHandler]"),
	}
	return h
}

func (h *ClientConnectionHandler) HandleConnectionEstablished(conn *connection.WsConnection) {
	loggerPrefix := fmt.Sprintf("[conn-%s]", conn.Address())
	rawConn := common_connection.NewConnection(context.Ctx.Logger().WithPrefix(loggerPrefix), conn, common_connection.DefaultTimeout, context.Ctx.MessageParser(), context.Ctx.NotificationEmitter())
	p := context.Ctx.AsyncTaskPool().Schedule(func() {
		rawConn.ReadingLoop()
	})
	h.logger.Printf("new connection %s received", rawConn.Address())
	// any message from any connection needs to go through here
	rawConn.OnIncomingMessage(func(message *messages.Message) {
		h.messageDispatcher.Dispatch(message, rawConn)
	})
	rawClient := client.NewAnonymousClient(rawConn)
	h.anonymousClientManager.AcceptClient(rawClient.Address(), rawClient)
	p.Wait()
}
