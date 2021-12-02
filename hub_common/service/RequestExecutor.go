package service

type IRequestExecutor interface {
	Execute(IServiceRequest)
}
