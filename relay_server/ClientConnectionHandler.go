package relay_server

import (
	"fmt"
	"wsdk/common/connection"
	"wsdk/common/logger"
	common_connection "wsdk/relay_common/connection"
	"wsdk/relay_common/message_actions"
	"wsdk/relay_common/messages"
	"wsdk/relay_server/container"
	"wsdk/relay_server/context"
	"wsdk/relay_server/controllers/connection_manager"
)

type ClientConnectionHandler struct {
	messageDispatcher message_actions.IMessageDispatcher
	connectionManager connection_manager.IConnectionManager `$inject:""`
	logger            *logger.SimpleLogger
}

type IClientConnectionHandler interface {
	HandleConnectionEstablished(conn connection.IConnection)
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

func (h *ClientConnectionHandler) HandleConnectionEstablished(conn connection.IConnection) {
	loggerPrefix := fmt.Sprintf("[conn-%s]", conn.Address())
	wrappedConn := common_connection.NewConnection(
		context.Ctx.Logger().WithPrefix(loggerPrefix),
		conn,
		common_connection.DefaultTimeout,
		context.Ctx.MessageParser(),
		context.Ctx.NotificationEmitter())
	h.logger.Printf("new connection %s received", wrappedConn.Address())
	// any message from any connection needs to go through here
	wrappedConn.OnIncomingMessage(func(message *messages.Message) {
		h.messageDispatcher.Dispatch(message, wrappedConn)
	})
	h.connectionManager.Accept(wrappedConn)
	// no need to run this on a different goroutine since each new connection is on its own coroutine
	wrappedConn.ReadingLoop()
	h.logger.Printf("connection %s cycle done", conn.Address())
}
