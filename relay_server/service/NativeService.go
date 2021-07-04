package service

import (
	"errors"
	"strings"
	"wsdk/relay_common/service"
	"wsdk/relay_server/context"
)

type NativeService struct {
	*Service
	handler service.IServiceHandler
}

type INativeService interface {
	IService
	RegisterRoute(shortUri string, handler service.RequestHandler) (err error)
	UnregisterRoute(shortUri string) (err error)
}

func NewNativeService(id, description string, serviceType, accessType, exeType int) *NativeService {
	handler := service.NewServiceHandler()
	return &NativeService{
		NewService(id, description, context.Ctx.Server(), NewInternalServiceRequestExecutor(context.Ctx, handler), make([]string, 0), serviceType, accessType, exeType),
		handler,
	}
}

func (s *NativeService) RegisterRoute(shortUri string, handler service.RequestHandler) (err error) {
	if strings.HasPrefix(shortUri, s.uriPrefix) {
		shortUri = strings.TrimPrefix(shortUri, s.uriPrefix)
	}
	s.withWrite(func() {
		s.serviceUris = append(s.serviceUris, shortUri)
		err = s.handler.Register(shortUri, handler)
	})
	return
}

func (s *NativeService) UnregisterRoute(shortUri string) (err error) {
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
