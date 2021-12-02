package connection_manager

import (
	base_conn "whub/common/connection"
	"whub/hub_common/connection"
	base_middleware "whub/hub_common/middleware"
	"whub/hub_common/service"
	"whub/hub_server/middleware"
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
