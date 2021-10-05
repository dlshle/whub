package auth

import (
	base_conn "wsdk/common/connection"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/service"
	"wsdk/relay_server/container"
	"wsdk/relay_server/middleware"
)

const (
	AuthMiddlewareId       = "auth"
	IsAuthorizedContextKey = "is_authorized"
	AuthToken              = "token"
	AuthMiddlewarePriority = 3
)

type AuthMiddleware struct {
	*middleware.ServerMiddleware
	authController IAuthModule `$inject:""`
}

func (m *AuthMiddleware) Init() error {
	m.ServerMiddleware = middleware.NewServerMiddleware(AuthMiddlewareId, AuthMiddlewarePriority)
	err := container.Container.Fill(m)
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
