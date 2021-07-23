package relay_client

import (
	"wsdk/relay_common/roles"
)

type ServerService struct {
	id       string
	host     string
	provider roles.IDescribableRole
	// TODO some service_manager descriptions here and a connection to send requests
}
