package error

type IControllerError interface {
	Code() int
	Error() string
}

type ControllerError struct {
	code int
	msg  string
}

func (e *ControllerError) Code() int {
	return e.code
}

func (e *ControllerError) Error() string {
	return e.msg
}

func NewControllerError(code int, msg string) IControllerError {
	return &ControllerError{
		code,
		msg,
	}
}
