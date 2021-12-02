package middleware

import (
	"whub/hub_common/connection"
	"whub/hub_common/service"
)

type RequestMiddleware func(conn connection.IConnection, request service.IServiceRequest) service.IServiceRequest
