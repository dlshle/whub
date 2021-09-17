package socket

import (
	"fmt"
	"net/http"
	"sync"
	"wsdk/common/connection"
	"wsdk/common/logger"
	common_connection "wsdk/relay_common/connection"
	"wsdk/relay_common/message_actions"
	"wsdk/relay_common/messages"
	"wsdk/relay_server/container"
	"wsdk/relay_server/context"
	"wsdk/relay_server/core/auth"
	"wsdk/relay_server/core/connection_manager"
	upgrader_util "wsdk/relay_server/http"
)

type ClientConnectionHandler struct {
	messageDispatcher message_actions.IMessageDispatcher
	connectionManager connection_manager.IConnectionManager `$inject:""`
	authController    auth.IAuthController                  `$inject:""`
	logger            *logger.SimpleLogger
	connPool          *sync.Pool
}

type IClientConnectionHandler interface {
	HandleConnectionEstablished(conn connection.IConnection, r *http.Request)
}

func NewClientConnectionHandler(messageDispatcher message_actions.IMessageDispatcher) IClientConnectionHandler {
	h := &ClientConnectionHandler{
		messageDispatcher: messageDispatcher,
		logger:            context.Ctx.Logger().WithPrefix("[ClientConnectionHandler]"),
		connPool: &sync.Pool{New: func() interface{} {
			return &common_connection.Connection{}
		}},
	}
	err := container.Container.Fill(h)
	if err != nil {
		panic(err)
	}
	return h
}

func (h *ClientConnectionHandler) HandleConnectionEstablished(conn connection.IConnection, r *http.Request) {
	loggerPrefix := fmt.Sprintf("[conn-%s]", conn.Address())
	wrappedConn := h.connPool.Get().(*common_connection.Connection)
	wrappedConn.Init(
		context.Ctx.Logger().WithPrefix(loggerPrefix),
		conn,
		common_connection.DefaultTimeout,
		context.Ctx.MessageParser(),
		context.Ctx.NotificationEmitter())
	h.logger.Printf("new connection %s received", wrappedConn.Address())
	// any message from any connection needs to go through here
	wrappedConn.OnIncomingMessage(func(message messages.IMessage) {
		h.messageDispatcher.Dispatch(message, wrappedConn)
	})
	h.connectionManager.AddConnection(wrappedConn)
	// should authorize the connection(register the connection to active client connection) when authorized
	clientId, err := h.authController.ValidateToken(upgrader_util.GetTokenFromQueryParameters(r))
	if err != nil {
		h.logger.Printf("unauthorized connection from %s", conn.Address())
		conn.Close()
		return
	} else {
		err = h.connectionManager.RegisterClientToConnection(clientId, conn.Address())
		if err != nil {
			h.logger.Printf("connection %s registration to client %s failed due to %s", conn.Address(), clientId, err.Error())
			conn.Close()
			return
		}
	}
	// no need to run this on a different goroutine since each new connection is on its own coroutine
	wrappedConn.ReadingLoop()
	h.logger.Printf("connection %s cycle done", conn.Address())
	h.connPool.Put(wrappedConn)
}
