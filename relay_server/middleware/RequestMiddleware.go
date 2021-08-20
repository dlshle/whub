package middleware

import (
	"wsdk/relay_common/connection"
	"wsdk/relay_common/service"
)

type RequestMiddleware func(conn connection.IConnection, request service.IServiceRequest) service.IServiceRequest
