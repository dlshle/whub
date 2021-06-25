package service

type IRequestExecutor interface {
	Execute(*ServiceRequest)
}
