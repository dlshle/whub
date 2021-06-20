package messages

import "wsdk/relay_common/connection"

type IMessageDispatcher interface {
	Dispatch(message *Message, conn *connection.WRConnection)
}
