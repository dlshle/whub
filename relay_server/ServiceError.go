package relay_server

import "fmt"

const (
	ErrInvalidServiceStatus = 11
	ErrInvalidServiceStatusTransition = 12
	ErrInvalidServiceMessageUri = 13
	ErrCanNotFindService = 14
)

func NewInvalidServiceStatusError(serviceId string, status int, reason string) IServerError {
	return NewServerError(ErrInvalidServiceStatusTransition, fmt.Sprintf("invalid service status %d of service(%s) due to %s", status, serviceId, reason))
}

func NewInvalidServiceStatusTransitionError(serviceId string, currentStatus int, newStatus int) IServerError {
	return NewServerError(ErrInvalidServiceStatusTransition, fmt.Sprintf("invalid service transition of service(%s) from %d to %d", serviceId, currentStatus, newStatus))
}

func NewInvalidServiceMessageUriError(uri string) IServerError {
	return NewServerError(ErrInvalidServiceMessageUri, fmt.Sprintf("invalid service message uri %s", uri))
}

func NewCanNotFindServiceError(uri string) IServerError {
	return NewServerError(ErrCanNotFindService, fmt.Sprintf("can not find service for uri %s", uri))
}