package auth_service

import (
	"encoding/json"
	"wsdk/common/logger"
	"wsdk/relay_common/service"
	"wsdk/relay_server/container"
	"wsdk/relay_server/core/auth"
	"wsdk/relay_server/service_base"
)

const (
	ID                 = "auth"
	RouteValidateToken = "/token"  // POST with token
	RouteLogin         = "/login"  // POST with id and password
	RouteLogout        = "/logout" // revoke my token
	RouteDelete        = "/delete" // delete account
)

type AuthService struct {
	*service_base.NativeService
	authController auth.IAuthController `$inject:""`
	logger         *logger.SimpleLogger
}

type TokenPayload struct {
	Token string `json:"token"`
}

func (s *AuthService) Init() (err error) {
	s.NativeService = service_base.NewNativeService(ID, "basic auth services", service.ServiceTypeInternal, service.ServiceAccessTypeBoth, service.ServiceExecutionSync)
	err = container.Container.Fill(s)
	if err != nil {
		return err
	}
	routeMap := make(map[string]service.RequestHandler)
	routeMap[RouteValidateToken] = s.ValidateToken
	return s.InitRoutes(routeMap)
}

func (s *AuthService) ValidateToken(request service.IServiceRequest, pathParams map[string]string, queryParams map[string]string) (err error) {
	var token TokenPayload
	from := request.From()
	if from == "" {
		return s.ResolveByInvalidCredential(request)
	}
	err = json.Unmarshal(request.Payload(), &token)
	if err != nil {
		return
	}
	clientId, err := s.authController.ValidateToken(token.Token)
	if err != nil || request.From() != clientId {
		return s.ResolveByInvalidCredential(request)
	}
	return
}
