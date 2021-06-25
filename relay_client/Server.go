package relay_client

import (
	"wsdk/relay_common/connection"
	"wsdk/relay_common/roles"
)

type Server struct {
	*roles.CommonServer
	*connection.Connection
}

type IServer interface {
	roles.ICommonServer
	connection.IConnection
}

// TODO impl
