package messages

type NextMessageHandler func(*Message)(*Message, error)
type HandlerFunction func(*Message, NextMessageHandler) (*Message, error)

type IMessageHandler interface {
	Handle(*Message, NextMessageHandler) (*Message, error)
}