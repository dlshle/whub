package client_management

import (
	"errors"
	"fmt"
	"wsdk/relay_common/roles"
	"wsdk/relay_common/service"
	"wsdk/relay_server/client"
	"wsdk/relay_server/container"
	"wsdk/relay_server/core/client_manager"
	"wsdk/relay_server/core/connection_manager"
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
	routeMap[RouteUpdate] = s.Update
	// TODO
	return s.InitRoutes(routeMap)
}

func (s *ClientManagementService) validateClientIdentity(request service.IServiceRequest) error {
	// TODO
	panic("implement me")
}

func (s *ClientManagementService) SignUp(request service.IServiceRequest, pathParams map[string]string, queryParams map[string]string) (err error) {
	// body should be client descriptor
	roleDesc, extraDesc, err := client_manager.UnmarshallClientDescriptor(request.Message())
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

func (s *ClientManagementService) Login(request service.IServiceRequest, pathParams map[string]string, queryParams map[string]string) (err error) {
	panic("implement")
}

func (s *ClientManagementService) Update(request service.IServiceRequest, pathParams map[string]string, queryParams map[string]string) (err error) {
	from := request.From()
	roleDesc, extraDesc, err := client_manager.UnmarshallClientDescriptor(request.Message())
	if err != nil {
		return err
	}
	if from != roleDesc.Id {
		err = errors.New(fmt.Sprintf("mismatch identity from(%s):desc(%s)", from, roleDesc.Id))
		s.Logger().Printf(err.Error())
		return err
	}
	client := client.NewClientFromDescriptor(roleDesc, extraDesc)
	err = s.clientManager.UpdateClient(client)
	if err != nil {
		s.Logger().Printf("error while updating client info due to %s", err.Error())
		return err
	}
	s.ResolveByResponse(request, ([]byte)(client.Describe().String()))
	return nil
}

func (s *ClientManagementService) GetById(request service.IServiceRequest, pathParams map[string]string, queryParams map[string]string) (err error) {
	panic("implement me")
}
