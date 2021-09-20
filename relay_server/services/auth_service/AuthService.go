package auth_service

import (
	"encoding/json"
	"errors"
	"wsdk/common/connection"
	"wsdk/common/logger"
	"wsdk/relay_common/messages"
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
	return s.InitHandlers(service.NewRequestHandlerMapBuilder().
		Post(RouteLogin, s.Login).
		Post(RouteValidateToken, s.ValidateToken).
		Post(RouteLogout, s.Logout).Build())
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

func (s *AuthService) Login(request service.IServiceRequest, pathParams map[string]string, queryParams map[string]string) (err error) {
	if request.From() != "" {
		return s.ResolveByError(request, messages.MessageTypeSvcForbiddenError, "you have already logged in")
	}
	loginModel, err := UnmarshallLoginPayload(request.Payload())
	if err != nil {
		return err
	}
	token, err := s.authController.Login(connection.TypeHTTP, loginModel.Id, loginModel.Password)
	if err != nil {
		return err
	}
	s.ResolveByResponse(request, MarshallLoginResponse(token))
	return nil
}

func (s *AuthService) Logout(request service.IServiceRequest, pathParams map[string]string, queryParams map[string]string) (err error) {
	if request.From() == "" {
		return s.ResolveByInvalidCredential(request)
	}
	cToken := request.GetContext(auth.AuthToken)
	if cToken == nil {
		return s.ResolveByInvalidCredential(request)
	}
	token, ok := cToken.(string)
	if !ok {
		return errors.New("can not cast token to string")
	}
	err = s.authController.RevokeToken(token)
	if err != nil {
		s.ResolveByResponse(request, ([]byte)("token has been revoked"))
	}
	return
}
