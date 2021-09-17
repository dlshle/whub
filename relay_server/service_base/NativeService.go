package service_base

import (
	"errors"
	"fmt"
	"strings"
	"wsdk/common/utils"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/service"
	"wsdk/relay_server/context"
	"wsdk/relay_server/request"
)

type NativeService struct {
	*Service
	handler service.ISimpleRequestHandler
}

type INativeService interface {
	IService
	RegisterRoute(shortUri string, handler service.RequestHandler) (err error)
	InitRoutes(routes map[string]service.RequestHandler) (err error)
	UnregisterRoute(shortUri string) (err error)
	ResolveByAck(request service.IServiceRequest) error
	ResolveByResponse(request service.IServiceRequest, responseData []byte) error
	ResolveByError(request service.IServiceRequest, errType int, msg string) error
	ResolveByInvalidCredential(request service.IServiceRequest) error
	Init() error
}

func NewNativeService(id, description string, serviceType, accessType, exeType int) *NativeService {
	handler := service.NewSimpleServiceHandler()
	return &NativeService{
		NewService(id, description, context.Ctx.Server(), request.NewInternalServiceRequestExecutor(handler), make([]string, 0), serviceType, accessType, exeType),
		handler,
	}
}

func (s *NativeService) RegisterRoute(uri string, handler service.RequestHandler) (err error) {
	defer s.Logger().Println(uri, "registration result: ", utils.ConditionalPick(err != nil, err, "success"))
	shortUri := uri
	if strings.HasPrefix(uri, s.uriPrefix) {
		shortUri = strings.TrimPrefix(uri, s.uriPrefix)
	}
	s.Logger().Println("registering new route: ", shortUri)
	s.withWrite(func() {
		s.serviceUris = append(s.serviceUris, shortUri)
		// handler needs full uri because service manager will provide full uri in request context
		err = s.handler.Register(fmt.Sprintf("%s%s", s.UriPrefix(), shortUri), handler)
	})
	return
}

func (s *NativeService) UnregisterRoute(shortUri string) (err error) {
	defer s.Logger().Println("route un-registration result: ", utils.ConditionalPick(err != nil, err, "success"))
	if strings.HasPrefix(shortUri, s.uriPrefix) {
		shortUri = strings.TrimPrefix(shortUri, s.uriPrefix)
	}
	s.Logger().Println("un-registering route: ", shortUri)
	uriIndex := -1
	for i, uri := range s.ServiceUris() {
		if uri == shortUri {
			uriIndex = i
		}
	}
	if uriIndex == -1 {
		return errors.New("shortUri " + shortUri + " does not exist")
	}
	s.withWrite(func() {
		l := len(s.serviceUris)
		s.serviceUris[l-1], s.serviceUris[uriIndex] = s.serviceUris[uriIndex], s.serviceUris[l-1]
		s.serviceUris = s.serviceUris[:l-1]
		err = s.handler.Unregister(fmt.Sprintf("%s%s", s.UriPrefix(), shortUri))
	})
	return
}

func (s *NativeService) CheckCredential(request service.IServiceRequest) error {
	if request.From() == "" {
		return errors.New("invalid credential")
	}
	return nil
}

func (s *NativeService) ResolveByAck(request service.IServiceRequest) error {
	return request.Resolve(messages.NewACKMessage(request.Id(), s.HostInfo().Id, request.From(), request.Uri()))
}

func (s *NativeService) ResolveByResponse(request service.IServiceRequest, responseData []byte) error {
	return request.Resolve(messages.NewMessage(request.Id(), s.HostInfo().Id, request.From(), request.Uri(), messages.MessageTypeSvcResponseOK, responseData))
}

func (s *NativeService) ResolveByError(request service.IServiceRequest, errType int, msg string) error {
	if errType < 400 || errType > 500 {
		return errors.New("invalid error code")
	}
	return request.Resolve(messages.NewMessage(request.Id(), s.HostInfo().Id, request.From(), request.Uri(), errType, s.assembleErrorMessageData(msg)))
}

func (s *NativeService) ResolveByInvalidCredential(request service.IServiceRequest) error {
	return s.ResolveByError(request, messages.MessageTypeSvcUnauthorizedError, "invalid credential")
}

func (s *NativeService) Handle(request service.IServiceRequest) messages.IMessage {
	// internal(business) services use short uri
	if s.ServiceType() == service.ServiceTypeInternal && strings.HasPrefix(request.Uri(), s.uriPrefix) {
		request.SetMessage(request.Message().SetUri(strings.TrimPrefix(request.Uri(), s.uriPrefix)))
	}
	return s.Service.Handle(request)
}

func (s *Service) assembleErrorMessageData(message string) []byte {
	return ([]byte)(fmt.Sprintf("{\"message\": \"%s\"}", message))
}

func (s *NativeService) Init() error {
	return errors.New("current native service did not implement Init() interface")
}

func (s *NativeService) InitRoutes(routes map[string]service.RequestHandler) (err error) {
	for k, v := range routes {
		if err = s.RegisterRoute(k, v); err != nil {
			return err
		}
	}
	return nil
}
