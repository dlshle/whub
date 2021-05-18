package messages

type SimpleMessageHandler func(*Message)(*Message, error)
type NextMessageHandler SimpleMessageHandler
type HandlerFunction func(*Message, NextMessageHandler) (*Message, error)

type IMessageHandler interface {
	Handle(*Message, NextMessageHandler) (*Message, error)
}