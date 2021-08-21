package connection_manager

import (
	base_conn "wsdk/common/connection"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/service"
	"wsdk/relay_server/core/middleware_manager"
	"wsdk/relay_server/middleware"
)

const (
	ConnectionMiddlewareId       = "connection"
	ConnectionMiddlewarePriority = 0
	IsSyncConnContextKey         = "is_sync_conn"
)

type ConnectionMiddleware struct {
	*middleware.ServerMiddleware
}

func (m *ConnectionMiddleware) Init() error {
	m.ServerMiddleware = middleware.NewServerMiddleware(ConnectionMiddlewareId, ConnectionMiddlewarePriority)
	return nil
}

func (m *ConnectionMiddleware) Run(conn connection.IConnection, request service.IServiceRequest) service.IServiceRequest {
	request.SetContext(IsSyncConnContextKey, !base_conn.IsAsyncType(conn.ConnectionType()))
	return request
}

func init() {
	middleware_manager.RegisterMiddleware(new(ConnectionMiddleware))
}
