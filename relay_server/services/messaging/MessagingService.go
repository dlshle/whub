package messaging

import (
	"errors"
	"fmt"
	"strings"
	service_common "wsdk/relay_common/service"
	"wsdk/relay_server/client"
	"wsdk/relay_server/context"
	"wsdk/relay_server/service"
)

const (
	ID             = "messaging"
	RouteSend      = "/send"
	RouteBroadcast = "/broadcast"
)

type MessagingService struct {
	*service.NativeService
	client.IClientManager
}

func NewMessagingService(ctx *context.Context, manager client.IClientManager) *MessagingService {
	messagingService := &MessagingService{
		service.NewNativeService(ctx, ID, "basic messaging service", service_common.ServiceTypeInternal, service_common.ServiceAccessTypeSocket, service_common.ServiceExecutionSync),
		manager,
	}
	messagingService.RegisterRoute(RouteSend, messagingService.Send)
	messagingService.RegisterRoute(RouteBroadcast, messagingService.Broadcast)
	return messagingService
}

func (s *MessagingService) Send(request *service_common.ServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	recv := s.GetClient(request.Message.To())
	if recv == nil {
		return errors.New(fmt.Sprintf("client %s is not online", request.Message.To()))
	}
	return recv.Send(request.Message)
}

func (s *MessagingService) Broadcast(request *service_common.ServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
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
	return nil
}
