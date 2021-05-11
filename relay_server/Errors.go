package relay_server

import "fmt"

const (
	ErrNoSuchClient = 0
	ErrNoSuchService = 1
	ErrClientInsufficientPermission = 2
	ErrClientExceededMaxServiceCount = 3
)

type ServerError struct {
	code int
	msg string
}

type IServerError interface {
	Error() string
	Code() int
}

func (e *ServerError) Error() string {
	return e.msg
}

func (e *ServerError) Code() int {
	return e.code
}

func NewServerError(code int, msg string) *ServerError {
	return &ServerError{code, msg}
}

func NewNoSuchClientError(clientId string) IServerError {
	return NewServerError(ErrNoSuchClient, fmt.Sprintf("no such client(%s)", clientId))
}

func NewNoSuchServiceError(serviceId string) IServerError {
	return NewServerError(ErrNoSuchService, fmt.Sprintf("no such service(%s)", serviceId))
}

func NewClientExceededMaxServiceCountError(clientId string) IServerError {
	return NewServerError(ErrClientExceededMaxServiceCount, fmt.Sprintf("client(%s) has exceeded max service count %d", clientId, MaxServicePerClient))
}