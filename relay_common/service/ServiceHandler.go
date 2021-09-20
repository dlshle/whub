package service

// ServiceHandler handles service requests
type IServiceHandler interface {
	Register(uri string, handler RequestHandler) error
	Unregister(uri string) error
	Handle(request IServiceRequest) error
}
