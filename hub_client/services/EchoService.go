package services

import (
	"whub/hub_client"
	"whub/hub_common/messages"
	"whub/hub_common/roles"
	"whub/hub_common/service"
)

const EchoServiceID = "echo"
const (
	EchoServiceRouteEcho = "/echo"
)

type EchoService struct {
	hub_client.IClientService
}

func (s *EchoService) Init(server roles.ICommonServer) (err error) {
	defer func() {
		s.Logger().Println("service has been initiated with err ", err)
	}()
	s.IClientService = hub_client.NewClientService(EchoServiceID, "simply echo messages", service.ServiceAccessTypeBoth, service.ServiceExecutionSync, server)
	err = s.RegisterRoute(messages.MessageTypeServiceGetRequest, EchoServiceRouteEcho, s.Echo)
	return err
}

func (s *EchoService) Echo(request service.IServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	s.ResolveByResponse(request, request.Payload())
	return nil
}
