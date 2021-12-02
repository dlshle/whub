package middleware

import (
	"whub/hub_common/connection"
	"whub/hub_common/service"
)

const (
	ConnectionTypeContextKey = "connection_type"
)

func ConnectionTypeMiddleware(conn connection.IConnection, request service.IServiceRequest) service.IServiceRequest {
	request.SetContext(ConnectionTypeContextKey, conn.ConnectionType())
	return request
}
