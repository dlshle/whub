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
	"wsdk/relay_server/controllers/anonymous_client_manager"
	client2 "wsdk/relay_server/controllers/client_manager"
	"wsdk/websocket/connection"
)

type ClientConnectionHandler struct {
	messageDispatcher      message_actions.IMessageDispatcher
	clientManager          client2.IClientManager                           `$inject:""`
	anonymousClientManager anonymous_client_manager.IAnonymousClientManager `$inject:""`
	logger                 *logger.SimpleLogger
}

type IClientConnectionHandler interface {
	HandleConnectionEstablished(conn *connection.WsConnection)
}

func NewClientConnectionHandler(messageDispatcher message_actions.IMessageDispatcher) IClientConnectionHandler {
	h := &ClientConnectionHandler{
		messageDispatcher: messageDispatcher,
		logger:            context.Ctx.Logger().WithPrefix("[ClientConnectionHandler]"),
	}
	err := container.Container.Fill(h)
	if err != nil {
		panic(err)
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
	h.logger.Printf("connection %s cycle done", conn.Address())
}
