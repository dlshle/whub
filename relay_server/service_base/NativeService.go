package service_base

import (
	"errors"
	"fmt"
	"strings"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/service"
	"wsdk/relay_server/context"
	"wsdk/relay_server/request"
)

type NativeService struct {
	*Service
	handler service.IDefaultServiceHandler
}

type INativeService interface {
	IService
	RegisterRoute(shortUri string, handler service.RequestHandler) (err error)
	RegisterRouteV1(requestType int, uri string, handler service.RequestHandler) (err error)
	InitRoutes(routes map[string]service.RequestHandler) (err error)
	InitHandlers(handlerMap map[int]map[string]service.RequestHandler) (err error)
	UnregisterRoute(requestType int, shortUri string) (err error)
	ResolveByAck(request service.IServiceRequest) error
	ResolveByResponse(request service.IServiceRequest, responseData []byte) error
	ResolveByError(request service.IServiceRequest, errType int, msg string) error
	ResolveByInvalidCredential(request service.IServiceRequest) error
	Init() error
}

func NewNativeService(id, description string, serviceType, accessType, exeType int) *NativeService {
	handler := service.NewDefaultServiceHandler()
	return &NativeService{
		NewService(id, description, context.Ctx.Server(), request.NewInternalServiceRequestExecutor(handler), make([]string, 0), serviceType, accessType, exeType),
		handler,
	}
}

// TODO deperacate this later
func (s *NativeService) RegisterRoute(uri string, handler service.RequestHandler) (err error) {
	return s.RegisterRouteV1(messages.MessageTypeServiceRequest, uri, handler)
}

func (s *NativeService) RegisterRouteV1(requestType int, uri string, handler service.RequestHandler) (err error) {
	defer func() {
		if err == nil {
			s.logger.Printf("handler %d %s has registered", requestType, uri)
		} else {
			s.logger.Printf("handler %d %s has registration failed due to %s", requestType, uri, err.Error())
		}
	}()
	shortUri := uri
	if strings.HasPrefix(shortUri, s.uriPrefix) {
		shortUri = strings.TrimPrefix(shortUri, s.uriPrefix)
	}
	// remove the extra / in the end to better format request uri(our convention is to not have / at the end)
	if len(shortUri) > 0 && shortUri[len(shortUri)-1] == '/' {
		shortUri = shortUri[:len(shortUri)-1]
	}
	s.withWrite(func() {
		s.serviceUris = append(s.serviceUris, shortUri)
		// handler needs full uri because service manager will provide full uri in request context
		err = s.handler.Register(requestType, fmt.Sprintf("%s%s", s.UriPrefix(), shortUri), handler)
	})
	return
}

func (s *NativeService) UnregisterRoute(requestType int, shortUri string) (err error) {
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
		err = s.handler.Unregister(requestType, fmt.Sprintf("%s%s", s.UriPrefix(), shortUri))
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
	return
}

func (s *NativeService) InitHandlers(handlerMap map[int]map[string]service.RequestHandler) (err error) {
	for requestType, uriHandlerMap := range handlerMap {
		for uri, handler := range uriHandlerMap {
			if err = s.RegisterRouteV1(requestType, uri, handler); err != nil {
				return err
			}
			delete(uriHandlerMap, uri)
		}
		delete(handlerMap, requestType)
	}
	return
}
