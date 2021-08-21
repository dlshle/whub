package auth

import (
	"wsdk/relay_common/connection"
	"wsdk/relay_common/service"
	"wsdk/relay_server/container"
	"wsdk/relay_server/core/middleware_manager"
	"wsdk/relay_server/middleware"
)

const (
	AuthMiddlewareId       = "auth"
	IsAuthorizedContextKey = "is_authorized"
	AuthMiddlewarePriority = 1
)

type AuthMiddleware struct {
	*middleware.ServerMiddleware
	authController IAuthController `$inject:""`
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

func init() {
	middleware_manager.RegisterMiddleware(new(AuthMiddleware))
}