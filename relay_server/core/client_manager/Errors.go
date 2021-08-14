package client_manager

import (
	"fmt"
	"wsdk/relay_server/core"
)

const (
	ClientNotFound = 101
)

func NewClientNotFoundError(id string) core.IControllerError {
	return core.NewControllerError(ClientNotFound, fmt.Sprintf("can not find client by id %s", id))
}
