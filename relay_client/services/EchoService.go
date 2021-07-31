package services

import (
	"wsdk/relay_client"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/roles"
	"wsdk/relay_common/service"
)

const EchoServiceID = "echo"
const (
	EchoServiceRouteEcho = "/echo"
)

type EchoService struct {
	relay_client.IClientService
}

func (s *EchoService) Init(server roles.ICommonServer, serverConn connection.IConnection) error {
	s.IClientService = relay_client.NewClientService(EchoServiceID, server, serverConn)
	err := s.RegisterRoute(EchoServiceRouteEcho, s.Echo)
	return err
}

func (s *EchoService) Echo(request *service.ServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	s.ResolveByResponse(request, request.Payload())
	return nil
}
