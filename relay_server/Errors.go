package relay_server

import (
	"encoding/json"
	"fmt"
)

const (
	ErrNoSuchClient = 0
	ErrNoSuchService = 1
	ErrClientInsufficientPermission = 2
	ErrClientExceededMaxServiceCount = 3

	ErrServerCloseFailed = 4

	ErrInvalidMessage = 5
)

type ServerError struct {
	code int
	msg string
}

type IServerError interface {
	Error() string
	Code() int
	Json() string
}

func (e *ServerError) Error() string {
	return e.msg
}

func (e *ServerError) Code() int {
	return e.code
}

func (e *ServerError) Json() string {
	jsonErr, err := json.Marshal(*e)
	if err != nil {
		return fmt.Sprintf("{\"code\":\"%d\", \"message\":\"%s\"}", e.code, e.msg)
	}
	return (string)(jsonErr)
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

func NewServerCloseFailError(msg string) IServerError {
	return NewServerError(ErrServerCloseFailed, msg)
}

func NewInvalidMessageError() IServerError {
	return NewServerError(ErrInvalidMessage, "invalid message, please contact system admin for further information")
}