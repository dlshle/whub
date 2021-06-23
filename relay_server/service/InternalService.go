package service

import (
	"errors"
	"strings"
	"wsdk/relay_common/service"
	"wsdk/relay_server"
)

type InternalService struct {
	*ServerService
	handler service.IServiceHandler
}

type IInternalService interface {
	IServerService
	RegisterRoute(shortUri string, handler service.RequestHandler) (err error)
	UnregisterRoute(shortUri string) (err error)
}

func NewInternalService(ctx *relay_server.Context, id, description string, serviceType, accessType, exeType int) *InternalService {
	handler := service.NewServiceHandler()
	return &InternalService{
		NewService(ctx, id, description, ctx.Identity(), NewInternalServiceRequestExecutor(ctx, handler), make([]string, 0), serviceType, accessType, exeType),
		handler,
	}
}

func (s *InternalService) RegisterRoute(shortUri string, handler service.RequestHandler) (err error) {
	if strings.HasPrefix(shortUri, s.uriPrefix) {
		shortUri = strings.TrimPrefix(shortUri, s.uriPrefix)
	}
	s.withWrite(func() {
		s.serviceUris = append(s.serviceUris, shortUri)
		err = s.handler.Register(shortUri, handler)
	})
	return
}

func (s *InternalService) UnregisterRoute(shortUri string) (err error) {
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
