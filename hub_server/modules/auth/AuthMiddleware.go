package auth

import (
	base_conn "whub/common/connection"
	"whub/hub_common/connection"
	"whub/hub_common/service"
	"whub/hub_server/middleware"
	"whub/hub_server/module_base"
)

const (
	AuthMiddlewareId       = "auth"
	IsAuthorizedContextKey = "is_authorized"
	AuthToken              = "token"
	AuthMiddlewarePriority = 3
)

type AuthMiddleware struct {
	*middleware.ServerMiddleware
	authController IAuthModule `module:""`
}

func (m *AuthMiddleware) Init() error {
	m.ServerMiddleware = middleware.NewServerMiddleware(AuthMiddlewareId, AuthMiddlewarePriority)
	err := module_base.Manager.AutoFill(m)
	if err != nil {
		panic(err)
	}
	return nil
}

func (m *AuthMiddleware) Run(conn connection.IConnection, request service.IServiceRequest) service.IServiceRequest {
	if !base_conn.IsAsyncType(conn.ConnectionType()) {
		request.SetContext(AuthToken, request.From())
	}
	clientId, err := m.authController.ValidateRequestSource(conn, request.Message())
	if err != nil {
		m.Logger().Printf("authentication failed due to %s", err.Error())
		request.SetFrom("")
		request.SetContext(IsAuthorizedContextKey, false)
	} else {
		request.SetFrom(clientId)
		request.SetContext(IsAuthorizedContextKey, clientId != "")
	}
	return request
}
