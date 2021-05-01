package WRCommon

type IRequestExecutor interface {
	Execute(message *Message) *ServiceMessage
}

type IRequestHandler interface {
	Handle(message *Message) *ServiceMessage
}

// RequestExecutor *-- RequestHandler
