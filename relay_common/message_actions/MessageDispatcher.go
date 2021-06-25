package message_actions

import (
	"wsdk/relay_common/connection"
	"wsdk/relay_common/messages"
)

type IMessageDispatcher interface {
	Dispatch(message *messages.Message, conn *connection.Connection)
}
