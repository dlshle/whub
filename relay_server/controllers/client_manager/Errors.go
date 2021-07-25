package client_manager

import (
	"fmt"
	"wsdk/relay_server/controllers"
)

const (
	ClientNotFound = 101
)

func NewClientNotFoundError(id string) controllers.IControllerError {
	return controllers.NewControllerError(ClientNotFound, fmt.Sprintf("can not find client by id %s", id))
}