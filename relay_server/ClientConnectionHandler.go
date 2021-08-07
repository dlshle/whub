package relay_server

import (
	"fmt"
	"wsdk/common/connection"
	"wsdk/common/logger"
	common_connection "wsdk/relay_common/connection"
	"wsdk/relay_common/message_actions"
	"wsdk/relay_common/messages"
	"wsdk/relay_server/client"
	"wsdk/relay_server/container"
	"wsdk/relay_server/context"
	"wsdk/relay_server/controllers/anonymous_client_manager"
	"wsdk/relay_server/controllers/client_manager"
)

type ClientConnectionHandler struct {
	messageDispatcher      message_actions.IMessageDispatcher
	clientManager          client_manager.IClientManager                    `$inject:""`
	anonymousClientManager anonymous_client_manager.IAnonymousClientManager `$inject:""`
	logger                 *logger.SimpleLogger
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
	warpedConn := common_connection.NewConnection(
		context.Ctx.Logger().WithPrefix(loggerPrefix),
		conn,
		common_connection.DefaultTimeout,
		context.Ctx.MessageParser(),
		context.Ctx.NotificationEmitter())
	h.logger.Printf("new connection %s received", warpedConn.Address())
	// any message from any connection needs to go through here
	warpedConn.OnIncomingMessage(func(message *messages.Message) {
		h.messageDispatcher.Dispatch(message, warpedConn)
	})
	rawClient := client.NewAnonymousClient(warpedConn)
	h.anonymousClientManager.AcceptClient(rawClient.Address(), rawClient)
	// no need to run this on a different goroutine since each new connection is on its own coroutine
	warpedConn.ReadingLoop()
	h.logger.Printf("connection %s cycle done", conn.Address())
}
