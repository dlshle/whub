package client_manager

import (
	"fmt"
	"wsdk/relay_server/core"
)

const (
	ClientNotFound  = 101
	InvalidClientId = 102
)

func NewClientNotFoundError(id string) core.IControllerError {
	return core.NewControllerError(ClientNotFound, fmt.Sprintf("can not find client by id %s", id))
}

func NewInvalidClientIdError(invalidId string) core.IControllerError {
	return core.NewControllerError(InvalidClientId, fmt.Sprintf("invalid client id %s", invalidId))
}
