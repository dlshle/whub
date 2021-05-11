package relay_server

import "fmt"

const (
	ErrInvalidServiceStatus = 11
	ErrInvalidServiceStatusTransition = 12
)

func NewInvalidServiceStatusError(serviceId string, status int, reason string) IServerError {
	return NewServerError(ErrInvalidServiceStatusTransition, fmt.Sprintf("invalid service status %d of service(%s) due to %s", status, serviceId, reason))
}

func NewInvalidServiceStatusTransitionError(serviceId string, currentStatus int, newStatus int) IServerError {
	return NewServerError(ErrInvalidServiceStatusTransition, fmt.Sprintf("invalid service transition of service(%s) from %d to %d", serviceId, currentStatus, newStatus))
}
