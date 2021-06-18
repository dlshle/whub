package service

import (
	"fmt"
	"wsdk/relay_server"
)

const (
	ErrInvalidServiceStatus           = 11
	ErrInvalidServiceStatusTransition = 12
	ErrInvalidServiceRequestUri       = 13
	ErrCanNotFindService              = 14
)

func NewInvalidServiceStatusError(serviceId string, status int, reason string) relay_server.IServerError {
	return relay_server.NewServerError(ErrInvalidServiceStatusTransition, fmt.Sprintf("invalid service status %d of service(%s) due to %s", status, serviceId, reason))
}

func NewInvalidServiceStatusTransitionError(serviceId string, currentStatus int, newStatus int) relay_server.IServerError {
	return relay_server.NewServerError(ErrInvalidServiceStatusTransition, fmt.Sprintf("invalid service transition of service(%s) from %d to %d", serviceId, currentStatus, newStatus))
}

func NewInvalidServiceRequestUriError(uri string) relay_server.IServerError {
	return relay_server.NewServerError(ErrInvalidServiceRequestUri, fmt.Sprintf("invalid service message uri %s", uri))
}

func NewCanNotFindServiceError(uri string) relay_server.IServerError {
	return relay_server.NewServerError(ErrCanNotFindService, fmt.Sprintf("can not find service for uri %s", uri))
}
