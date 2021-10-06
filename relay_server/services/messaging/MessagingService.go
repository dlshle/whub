package messaging

import (
	"errors"
	"fmt"
	"strings"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/service"
	"wsdk/relay_server/client"
	"wsdk/relay_server/module_base"
	client_manager "wsdk/relay_server/modules/client_manager"
	"wsdk/relay_server/modules/connection_manager"
	"wsdk/relay_server/service_base"
)

const (
	ID             = "message"
	RouteSend      = "/send"
	RouteBroadcast = "/broadcast"
)

type MessagingService struct {
	*service_base.NativeService
	client_manager.IClientManagerModule `module:""`
	connManager                         connection_manager.IConnectionManagerModule `module:""`
	// logger *logger.SimpleLogger
}

func (s *MessagingService) Init() (err error) {
	s.NativeService = service_base.NewNativeService(ID, "basic messaging service", service.ServiceTypeInternal, service.ServiceAccessTypeSocket, service.ServiceExecutionSync)
	err = module_base.Manager.AutoFill(s)
	if err != nil {
		return err
	}
	if s.IClientManagerModule == nil {
		return errors.New("can not get clientManager from container")
	}
	return s.RegisterRoutes(service.NewRequestHandlerMapBuilder().
		Post(RouteSend, s.Send).
		Post(RouteBroadcast, s.Broadcast).
		Build())
}

func (s *MessagingService) sendToClient(id string, message messages.IMessage, onSendErr func(err error)) error {
	conns, err := s.connManager.GetConnectionsByClientId(id)
	if err != nil {
		return err
	}
	if len(conns) == 0 {
		return errors.New(fmt.Sprintf("unable to send message to %s because the client is not online", id))
	}
	for _, conn := range conns {
		err = conn.Send(message)
		if err != nil && onSendErr != nil {
			onSendErr(err)
		}
	}
	return nil
}

func (s *MessagingService) Send(request service.IServiceRequest, pathParams map[string]string, queryParams map[string]string) (err error) {
	if _, err = s.GetClientWithErrOnNotFound(request.From()); err != nil {
		return err
	}
	err = s.sendToClient(request.From(), request.Message(), func(cerr error) {
		err = cerr
	})
	if err != nil {
		return err
	}
	s.ResolveByAck(request)
	return nil
}

func (s *MessagingService) Broadcast(request service.IServiceRequest, pathParams map[string]string, queryParams map[string]string) (err error) {
	defer s.Logger().Printf("%s broadcast result: %v", request.Message().String(), err)
	errMsg := strings.Builder{}
	s.WithAllClients(func(clients []*client.Client) {
		for _, c := range clients {
			err = s.sendToClient(c.Id(), request.Message(), func(cerr error) {
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
