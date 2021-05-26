package messages

type MessageHandlerFunc func(*Message)(*Message, error)
type NextMessageHandler MessageHandlerFunc
type HandlerFunction func(*Message, NextMessageHandler) (*Message, error)

type IMessageHandler interface {
	Handle(*Message, NextMessageHandler) (*Message, error)
}