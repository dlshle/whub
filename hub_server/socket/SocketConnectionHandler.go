package socket

import (
	"fmt"
	"net/http"
	"sync"
	"whub/common/connection"
	"whub/common/logger"
	common_connection "whub/hub_common/connection"
	"whub/hub_common/dispatcher"
	"whub/hub_common/messages"
	"whub/hub_server/context"
	upgrader_util "whub/hub_server/http"
	"whub/hub_server/module_base"
	"whub/hub_server/modules/auth"
	"whub/hub_server/modules/connection_manager"
)

type SocketConnectionHandler struct {
	messageDispatcher dispatcher.IMessageDispatcher
	connectionManager connection_manager.IConnectionManagerModule `module:""`
	authController    auth.IAuthModule                            `module:""`
	logger            *logger.SimpleLogger
	connPool          *sync.Pool
}

type ISocketConnectionHandler interface {
	HandleConnectionEstablished(conn connection.IConnection, r *http.Request)
}

func NewSocketConnectionHandler(messageDispatcher dispatcher.IMessageDispatcher) ISocketConnectionHandler {
	h := &SocketConnectionHandler{
		messageDispatcher: messageDispatcher,
		logger:            context.Ctx.Logger().WithPrefix("[SocketConnectionHandler]"),
		connPool: &sync.Pool{New: func() interface{} {
			return &common_connection.Connection{}
		}},
	}
	err := module_base.Manager.AutoFill(h)
	if err != nil {
		panic(err)
	}
	return h
}

func (h *SocketConnectionHandler) HandleConnectionEstablished(conn connection.IConnection, r *http.Request) {
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
