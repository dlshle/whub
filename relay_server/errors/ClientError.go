package errors

import "fmt"

const (
	ErrClientAlreadyConnected = 21
	ErrClientNotConnected     = 22
	ErrCanNotFindClientByAddr = 23
)

func NewClientAlreadyConnectedError(clientId string) IServerError {
	return NewServerError(ErrClientAlreadyConnected, fmt.Sprintf("client %s has already connected", clientId))
}

func NewClientNotConnectedError(clientId string) IServerError {
	return NewServerError(ErrClientNotConnected, fmt.Sprintf("client %s is not connected", clientId))
}

func NewCanNotFindClientByAddr(addr string) IServerError {
	return NewServerError(ErrClientNotConnected, fmt.Sprintf("can not find client by address %s", addr))
}
