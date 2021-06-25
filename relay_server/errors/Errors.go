package errors

import (
	"encoding/json"
	"fmt"
)

type ServerError struct {
	code int
	msg  string
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
