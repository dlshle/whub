package service_base

import (
	"fmt"
	"wsdk/relay_server/errors"
)

const (
	ErrInvalidServiceStatus           = 11
	ErrInvalidServiceStatusTransition = 12
	ErrInvalidServiceRequestUri       = 13
	ErrCanNotFindService              = 14
)

func NewInvalidServiceStatusError(serviceId string, status int, reason string) errors.IServerError {
	return errors.NewServerError(ErrInvalidServiceStatusTransition, fmt.Sprintf("invalid service_manager status %d of service_manager(%s) due to %s", status, serviceId, reason))
}

func NewInvalidServiceStatusTransitionError(serviceId string, currentStatus int, newStatus int) errors.IServerError {
	return errors.NewServerError(ErrInvalidServiceStatusTransition, fmt.Sprintf("invalid service_manager transition of service_manager(%s) from %d to %d", serviceId, currentStatus, newStatus))
}

func NewInvalidServiceRequestUriError(uri string) errors.IServerError {
	return errors.NewServerError(ErrInvalidServiceRequestUri, fmt.Sprintf("invalid service_manager message uri_trie %s", uri))
}

func NewCanNotFindServiceError(uri string) errors.IServerError {
	return errors.NewServerError(ErrCanNotFindService, fmt.Sprintf("can not find service_manager for uri_trie %s", uri))
}
