package middleware

import (
	"wsdk/relay_common/connection"
	"wsdk/relay_common/service"
)

const (
	ConnectionTypeContextKey = "connection_type"
)

func ConnectionTypeMiddleware(conn connection.IConnection, request service.IServiceRequest) service.IServiceRequest {
	request.SetContext(ConnectionTypeContextKey, conn.ConnectionType())
	return request
}
