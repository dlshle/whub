package service_base

import (
	"errors"
	"strings"
	"wsdk/common/utils"
	"wsdk/relay_common/service"
	"wsdk/relay_server/context"
	"wsdk/relay_server/request"
)

type NativeService struct {
	*Service
	handler service.IServiceHandler
}

type INativeService interface {
	IService
	RegisterRoute(shortUri string, handler service.RequestHandler) (err error)
	UnregisterRoute(shortUri string) (err error)
	Init() error
}

func NewNativeService(id, description string, serviceType, accessType, exeType int) *NativeService {
	handler := service.NewServiceHandler()
	return &NativeService{
		NewService(id, description, context.Ctx.Server(), request.NewInternalServiceRequestExecutor(handler), make([]string, 0), serviceType, accessType, exeType),
		handler,
	}
}

func (s *NativeService) RegisterRoute(shortUri string, handler service.RequestHandler) (err error) {
	defer s.Logger().Println(shortUri, "registration result: ", utils.ConditionalPick(err != nil, err, "success"))
	if strings.HasPrefix(shortUri, s.uriPrefix) {
		shortUri = strings.TrimPrefix(shortUri, s.uriPrefix)
	}
	s.Logger().Println("registering new route: ", shortUri)
	s.withWrite(func() {
		s.serviceUris = append(s.serviceUris, shortUri)
		err = s.handler.Register(shortUri, handler)
	})
	return
}

func (s *NativeService) UnregisterRoute(shortUri string) (err error) {
	defer s.Logger().Println("route un-registration result: ", utils.ConditionalPick(err != nil, err, "success"))
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
		err = s.handler.Unregister(shortUri)
	})
	return
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
