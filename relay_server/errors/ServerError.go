package errors

import (
	"fmt"
)

const (
	ErrNoSuchClient                  = 0
	ErrNoSuchService                 = 1
	ErrClientInsufficientPermission  = 2
	ErrClientExceededMaxServiceCount = 3

	ErrServerCloseFailed = 4

	ErrInvalidMessage = 5
)

func NewServerError(code int, msg string) *ServerError {
	return &ServerError{code, msg}
}

func NewNoSuchClientError(clientId string) IServerError {
	return NewServerError(ErrNoSuchClient, fmt.Sprintf("no such client_manager(%s)", clientId))
}

func NewNoSuchServiceError(serviceId string) IServerError {
	return NewServerError(ErrNoSuchService, fmt.Sprintf("no such service_manager(%s)", serviceId))
}

func NewClientExceededMaxServiceCountError(clientId string, maxServicePerClient int) IServerError {
	return NewServerError(ErrClientExceededMaxServiceCount, fmt.Sprintf("client_manager(%s) has exceeded max service_manager count %d", clientId, maxServicePerClient))
}

func NewServerCloseFailError(msg string) IServerError {
	return NewServerError(ErrServerCloseFailed, msg)
}

func NewInvalidMessageError() IServerError {
	return NewServerError(ErrInvalidMessage, "invalid message, please contact system admin for further information")
}
