package client_manager

import (
	"fmt"
	error2 "wsdk/relay_server/core/error"
)

const (
	ClientNotFound  = 101
	InvalidClientId = 102
)

func NewClientNotFoundError(id string) error2.IControllerError {
	return error2.NewControllerError(ClientNotFound, fmt.Sprintf("can not find client by id %s", id))
}

func NewInvalidClientIdError(invalidId string) error2.IControllerError {
	return error2.NewControllerError(InvalidClientId, fmt.Sprintf("invalid client id %s", invalidId))
}
