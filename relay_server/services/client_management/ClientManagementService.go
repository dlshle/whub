package client_management

import (
	"encoding/json"
	"errors"
	"fmt"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/roles"
	"wsdk/relay_common/service"
	"wsdk/relay_server/client"
	"wsdk/relay_server/container"
	"wsdk/relay_server/modules/client_manager"
	"wsdk/relay_server/modules/connection_manager"
	"wsdk/relay_server/service_base"
	"wsdk/relay_server/services/auth_service"
)

const (
	ID                  = "clients"
	RouteSignUp         = "/signup"
	RouteDelete         = "/"
	RouteGet            = "/:id"
	RouteGetAll         = "/"
	RouteUpdate         = "/"
	RouteGetConnections = "/:id/conn"
)

type ClientManagementService struct {
	*service_base.NativeService
	clientManager client_manager.IClientManagerModule         `$inject:""`
	connManager   connection_manager.IConnectionManagerModule `$inject:""`
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
	return s.InitHandlers(service.NewRequestHandlerMapBuilder().
		Post(RouteSignUp, s.SignUp).
		Put(RouteUpdate, s.Update).
		Get(RouteGet, s.GetById).
		Get(RouteGetAll, s.GetAll).
		Get(RouteGetConnections, s.GetConnections).
		Delete(RouteDelete, s.Delete).Build())
}

func (s *ClientManagementService) SignUp(request service.IServiceRequest, pathParams map[string]string, queryParams map[string]string) (err error) {
	if request.From() != "" {
		return s.ResolveByError(request, messages.MessageTypeSvcBadRequestError, "you have already logged in")
	}
	// body should be client descriptor
	signupModel, err := auth_service.UnmarshallClientSignupModel(request.Payload())
	if err != nil {
		return err
	}
	client := client.NewClient(signupModel.Id, signupModel.Description, roles.ClientTypeAuthenticated, signupModel.Password, 0)
	err = s.clientManager.AddClient(client)
	if err != nil {
		return err
	}
	client.SetCKey("******")
	s.ResolveByResponse(request, ([]byte)(client.Describe().String()))
	return nil
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
		return s.ResolveByError(request, 403, err.Error())
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
	// one can only checks limited info of a client info except for super admin and oneself
	myId := request.From()
	isLoggedIn := myId != ""
	me := client.NewAnonymousClient()
	if isLoggedIn {
		me, err = s.getCurrentUser(myId)
		if err != nil {
			return err
		}
	}
	isAnonymous := true
	clientId := pathParams["id"]
	if clientId == myId || me.CType() > roles.ClientTypeAuthenticated {
		isAnonymous = false
	}
	client, err := s.clientManager.GetClient(clientId)
	if err != nil {
		return err
	}
	s.ResolveByResponse(request, s.getMarshalledClientInfo(client, isAnonymous))
	return nil
}

func (s *ClientManagementService) getCurrentUser(id string) (*client.Client, error) {
	curr, err := s.clientManager.GetClient(id)
	if err != nil && curr == nil {
		curr = client.NewAnonymousClient()
	}
	return curr, err
}

func (s *ClientManagementService) getMarshalledClientInfo(client *client.Client, isAnonymous bool) []byte {
	clientDesc := client.Describe()
	if isAnonymous {
		clientDesc.ExtraInfo = ""
	}
	return ([]byte)(clientDesc.String())
}

func (s *ClientManagementService) Delete(request service.IServiceRequest, pathParams map[string]string, queryParams map[string]string) (err error) {
	me, err := s.getCurrentUser(request.From())
	if err != nil {
		return
	}
	if me.CType() < roles.ClientTypeManager {
		return s.ResolveByError(request, messages.MessageTypeSvcBadRequestError, "you are not authorized to delete clients")
	}
	deleteClientsPayload, err := UnmarshalDeleteClientsPayload(request.Payload())
	if err != nil {
		return
	}
	for _, id := range deleteClientsPayload.ids {
		err = s.clientManager.DeleteClient(id)
		if err != nil {
			return
		}
	}
	s.ResolveByResponse(request, ([]byte)("ok"))
	return nil
}

func (s *ClientManagementService) GetAll(request service.IServiceRequest, pathParams map[string]string, queryParams map[string]string) (err error) {
	myId := request.From()
	if myId == "" {
		return s.ResolveByError(request, messages.MessageTypeSvcBadRequestError, "invalid identity")
	}
	me, err := s.getCurrentUser(myId)
	if err != nil {
		return err
	}
	if me.CType() < roles.ClientTypeManager {
		return s.ResolveByError(request, messages.MessageTypeSvcBadRequestError, "insufficient privilege")
	}
	allClients, err := s.clientManager.GetAllClients()
	if err != nil {
		return err
	}
	described := make([]roles.RoleDescriptor, len(allClients), len(allClients))
	for i, c := range allClients {
		described[i] = c.Describe()
	}
	marshalled, err := json.Marshal(described)
	if err != nil {
		return err
	}
	s.ResolveByResponse(request, marshalled)
	return nil
}

func (s *ClientManagementService) GetConnections(request service.IServiceRequest, pathParams map[string]string, queryParams map[string]string) (err error) {
	from := request.From()
	clientId := pathParams["id"]
	if from == "" {
		return s.ResolveByError(request, messages.MessageTypeSvcBadRequestError, "invalid credential: you need to login to check connections")
	}
	me, err := s.getCurrentUser(from)
	if err != nil {
		return err
	}
	allowed := from == clientId || me.CType() > roles.ClientTypeAuthenticated
	if !allowed {
		return s.ResolveByError(request, messages.MessageTypeSvcBadRequestError, "insufficient privilege: you do not have access to such information")
	}
	conns, err := s.connManager.GetConnectionsByClientId(clientId)
	if err != nil {
		return err
	}
	strConns := make([]string, len(conns), len(conns))
	for i, c := range conns {
		strConns[i] = c.String()
	}
	marshalled, err := json.Marshal(strConns)
	if err != nil {
		return err
	}
	s.ResolveByResponse(request, marshalled)
	return nil
}
