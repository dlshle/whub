package messaging

import (
	"errors"
	"fmt"
	"strings"
	service_common "wsdk/relay_common/service"
	"wsdk/relay_server/client"
	"wsdk/relay_server/container"
	"wsdk/relay_server/managers"
	"wsdk/relay_server/service"
)

const (
	ID             = "message"
	RouteSend      = "/send"
	RouteBroadcast = "/broadcast"
)

type MessagingService struct {
	*service.NativeService
	managers.IClientManager
	// logger *logger.SimpleLogger
}

func New() service.INativeService {
	messagingService := &MessagingService{
		service.NewNativeService(ID, "basic messaging service", service_common.ServiceTypeInternal, service_common.ServiceAccessTypeSocket, service_common.ServiceExecutionSync),
		container.Container.GetById(managers.ClientManagerId).(managers.IClientManager),
	}
	messagingService.RegisterRoute(RouteSend, messagingService.Send)
	messagingService.RegisterRoute(RouteBroadcast, messagingService.Broadcast)
	return messagingService
}

func (s *MessagingService) Init() (err error) {
	s.NativeService = service.NewNativeService(ID, "basic messaging service", service_common.ServiceTypeInternal, service_common.ServiceAccessTypeSocket, service_common.ServiceExecutionSync)
	s.IClientManager = container.Container.GetById(managers.ClientManagerId).(managers.IClientManager)
	if s.IClientManager == nil {
		return errors.New("can not get clientManager from container")
	}
	err = s.RegisterRoute(RouteSend, s.Send)
	if err != nil {
		return
	}
	return s.RegisterRoute(RouteBroadcast, s.Broadcast)
}

func (s *MessagingService) Send(request *service_common.ServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	recv := s.GetClient(request.Message.To())
	if recv == nil {
		return errors.New(fmt.Sprintf("client %s is not online", request.Message.To()))
	}
	err := recv.Send(request.Message)
	if err != nil {
		return err
	}
	s.ResolveByAck(request)
	return nil
}

func (s *MessagingService) Broadcast(request *service_common.ServiceRequest, pathParams map[string]string, queryParams map[string]string) (err error) {
	defer s.Logger().Printf("%s broadcast result: %v", request.Message.String(), err)
	errMsg := strings.Builder{}
	s.WithAllClients(func(clients []*client.Client) {
		for _, c := range clients {
			err := c.Send(request.Message)
			if err != nil {
				errMsg.WriteString(err.Error())
				errMsg.WriteByte('\n')
			}
		}
	})
	if errMsg.Len() > 0 {
		return errors.New(errMsg.String())
	}
	s.ResolveByAck(request)
	return nil
}
