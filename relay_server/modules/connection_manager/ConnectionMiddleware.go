package connection_manager

import (
	base_conn "wsdk/common/connection"
	"wsdk/relay_common/connection"
	base_middleware "wsdk/relay_common/middleware"
	"wsdk/relay_common/service"
	"wsdk/relay_server/middleware"
)

const (
	ConnectionMiddlewareId       = "connection"
	ConnectionMiddlewarePriority = 0
	IsSyncConnContextKey         = "is_sync_conn"
	AddrContextKey               = "address"
)

type ConnectionMiddleware struct {
	*middleware.ServerMiddleware
}

func (m *ConnectionMiddleware) Init() error {
	m.ServerMiddleware = middleware.NewServerMiddleware(ConnectionMiddlewareId, ConnectionMiddlewarePriority)
	return nil
}

func (m *ConnectionMiddleware) Run(conn connection.IConnection, request service.IServiceRequest) service.IServiceRequest {
	request = base_middleware.ConnectionTypeMiddleware(conn, request)
	request.SetContext(IsSyncConnContextKey, !base_conn.IsAsyncType(conn.ConnectionType()))
	request.SetContext(AddrContextKey, conn.Address())
	return request
}