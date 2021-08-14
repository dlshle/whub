package client_management

import (
	"encoding/json"
	"wsdk/common/logger"
	"wsdk/common/utils"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/roles"
	"wsdk/relay_common/service"
	"wsdk/relay_server/client"
	"wsdk/relay_server/container"
	"wsdk/relay_server/controllers/client_manager"
	"wsdk/relay_server/controllers/connection_manager"
	"wsdk/relay_server/service_base"
)

const (
	ID                  = "client"
	RouteSignUp         = "/signup"
	RouteLogin          = "/login" // async only
	RouteSignOff        = "/signoff"
	RouteGet            = "/get/:id"
	RouteGetAll         = "/get"
	RouteUpdate         = "/update"
	RouteGetConnections = "/get/:id/conn"
)

type ClientManagementService struct {
	*service_base.NativeService
	clientManager client_manager.IClientManager         `$inject:""`
	connManager   connection_manager.IConnectionManager `$inject:""`
	logger        *logger.SimpleLogger
}

func (s *ClientManagementService) Init() (err error) {
	s.NativeService = service_base.NewNativeService(ID,
		"client information management",
		service.ServiceTypeInternal,
		service.ServiceAccessTypeBoth,
		service.ServiceExecutionBoth)
	err = container.Container.Fill(s)
	if err != nil {
		return err
	}
	routeMap := make(map[string]service.RequestHandler)
	routeMap[RouteSignUp] = s.SignUp
	// TODO
	return s.InitRoutes(routeMap)
}

func (s *ClientManagementService) unmarshallClientDescriptor(message *messages.Message) (roleDescriptor roles.RoleDescriptor, extraInfoDescriptor roles.ClientExtraInfoDescriptor, err error) {
	err = utils.ProcessWithError([]func() error{
		func() error {
			return json.Unmarshal(message.Payload(), &roleDescriptor)
		},
		func() error {
			return json.Unmarshal(([]byte)(roleDescriptor.ExtraInfo), &extraInfoDescriptor)
		},
	})
	return
}

func (s *ClientManagementService) SignUp(request *service.ServiceRequest, pathParams map[string]string, queryParams map[string]string) (err error) {
	// body should be client descriptor
	roleDesc, extraDesc, err := s.unmarshallClientDescriptor(request.Message)
	if err != nil {
		return err
	}
	err = s.clientManager.AddClient(client.NewClient(roleDesc.Id, roleDesc.Description, roles.ClientTypeAuthenticated, extraDesc.CKey, 0))
	if err != nil {
		return err
	}
	s.ResolveByResponse(request, ([]byte)(roleDesc.String()))
	return nil
}

func (s *ClientManagementService) Login(request *service.ServiceRequest, pathParams map[string]string, queryParams map[string]string) (err error) {
	panic("implement")
}
