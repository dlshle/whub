package messages

type IMessageDispatcher interface {
	Dispatch(message *Message) error
}
