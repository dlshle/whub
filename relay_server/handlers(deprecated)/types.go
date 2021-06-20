package handlers_deprecated_

import "wsdk/relay_common/messages"

type MessageHandlerFunc func(*messages.Message) (*messages.Message, error)
type NextMessageHandler MessageHandlerFunc
type HandlerFunction func(*messages.Message, NextMessageHandler) (*messages.Message, error)

type IMessageHandler interface {
	Handle(*messages.Message, NextMessageHandler) (*messages.Message, error)
}
