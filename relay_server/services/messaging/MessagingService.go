package messaging

import (
	"errors"
	"fmt"
	"strings"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/service"
	"wsdk/relay_server/client"
	"wsdk/relay_server/container"
	"wsdk/relay_server/controllers/client_manager"
	"wsdk/relay_server/controllers/connection_manager"
	"wsdk/relay_server/service_base"
)

const (
	ID             = "message"
	RouteSend      = "/send"
	RouteBroadcast = "/broadcast"
)

type MessagingService struct {
	*service_base.NativeService
	client_manager.IClientManager `$inject:""`
	connManager                   connection_manager.IConnectionManager `$inject:""`
	// logger *logger.SimpleLogger
}

func New() service_base.INativeService {
	messagingService := &MessagingService{
		NativeService: service_base.NewNativeService(ID, "basic messaging service", service.ServiceTypeInternal, service.ServiceAccessTypeSocket, service.ServiceExecutionSync),
	}
	err := container.Container.Fill(messagingService)
	if err != nil {
		messagingService.Logger().Println(err)
	}
	messagingService.RegisterRoute(RouteSend, messagingService.Send)
	messagingService.RegisterRoute(RouteBroadcast, messagingService.Broadcast)
	return messagingService
}

func (s *MessagingService) Init() (err error) {
	s.NativeService = service_base.NewNativeService(ID, "basic messaging service", service.ServiceTypeInternal, service.ServiceAccessTypeSocket, service.ServiceExecutionSync)
	err = container.Container.Fill(s)
	if err != nil {
		return err
	}
	if s.IClientManager == nil {
		return errors.New("can not get clientManager from container")
	}
	err = s.RegisterRoute(RouteSend, s.Send)
	if err != nil {
		return
	}
	return s.RegisterRoute(RouteBroadcast, s.Broadcast)
}

func (s *MessagingService) sendToClient(id string, message *messages.Message, onSendErr func(err error)) error {
	conns, err := s.connManager.GetConnectionsByClientId(id)
	if err != nil {
		return err
	}
	for _, conn := range conns {
		err = conn.Send(message)
		if err != nil && onSendErr != nil {
			onSendErr(err)
		}
	}
	return nil
}

func (s *MessagingService) Send(request *service.ServiceRequest, pathParams map[string]string, queryParams map[string]string) (err error) {
	recv := s.GetClient(request.Message.To())
	if recv == nil {
		return errors.New(fmt.Sprintf("client %s is not online", request.Message.To()))
	}
	err = s.sendToClient(recv.Id(), request.Message, func(cerr error) {
		err = cerr
	})
	if err != nil {
		return err
	}
	s.ResolveByAck(request)
	return nil
}

func (s *MessagingService) Broadcast(request *service.ServiceRequest, pathParams map[string]string, queryParams map[string]string) (err error) {
	defer s.Logger().Printf("%s broadcast result: %v", request.Message.String(), err)
	errMsg := strings.Builder{}
	s.WithAllClients(func(clients []*client.Client) {
		for _, c := range clients {
			err = s.sendToClient(c.Id(), request.Message, func(cerr error) {
				err = cerr
			})
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
