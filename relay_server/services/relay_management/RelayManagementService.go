package relay_management

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"wsdk/common/utils"
	"wsdk/relay_common/messages"
	service_common "wsdk/relay_common/service"
	"wsdk/relay_server/container"
	"wsdk/relay_server/controllers"
	servererror "wsdk/relay_server/errors"
	"wsdk/relay_server/events"
	"wsdk/relay_server/service"
	server_utils "wsdk/relay_server/utils"
)

const (
	ID                         = "relay"
	RouteRegisterService       = "/register"   // payload = service descriptor
	RouteUnregisterService     = "/unregister" // payload = service descriptor
	RouteUpdateService         = "/update"     // payload = service descriptor
	RouteGetAllServices        = "/services"   // need privilege, respond with all relayed services
	RouteGetServicesByClientId = "/services/:clientId"
)

type RelayManagementService struct {
	*service.NativeService
	clientManager  controllers.IClientManager  `$inject:""`
	serviceManager controllers.IServiceManager `$inject:""`
	servicePool    *sync.Pool
}

func (s *RelayManagementService) Init() error {
	s.NativeService = service.NewNativeService(ID, "relay management service", service_common.ServiceTypeInternal, service_common.ServiceAccessTypeSocket, service_common.ServiceExecutionSync)
	s.servicePool = &sync.Pool{
		New: func() interface{} {
			return new(service.RelayService)
		},
	}
	container.Container.Fill(s)
	if s.clientManager == nil {
		return errors.New("can not get clientManager from container")
	}
	if s.serviceManager == nil {
		return errors.New("can not get serviceManager from container")
	}
	return s.init()
}

func (s *RelayManagementService) init() error {
	s.initNotificationHandlers()
	return s.initRoutes()
}

func (s *RelayManagementService) initNotificationHandlers() {
	events.OnEvent(events.EventClientDisconnected, func(message *messages.Message) {
		clientId := string(message.Payload()[:])
		s.serviceManager.UnregisterAllServicesFromClientId(clientId)
	})
	events.OnEvent(events.EventClientUnexpectedClosure, func(message *messages.Message) {
		clientId := string(message.Payload()[:])
		s.serviceManager.WithServicesFromClientId(clientId, func(services []service.IService) {
			for _, svc := range services {
				svc.Kill()
			}
		})
	})
	events.OnEvent(events.EventClientConnected, func(message *messages.Message) {
		clientId := string(message.Payload()[:])
		s.tryToRestoreDeadServicesFromReconnectedClient(clientId)
	})
}

func (s *RelayManagementService) initRoutes() (err error) {
	err = s.RegisterRoute(RouteRegisterService, s.RegisterService)
	if err != nil {
		return
	}
	err = s.RegisterRoute(RouteUnregisterService, s.UnregisterService)
	if err != nil {
		return
	}
	err = s.RegisterRoute(RouteUpdateService, s.UpdateService)
	if err != nil {
		return
	}
	err = s.RegisterRoute(RouteGetServicesByClientId, s.GetServiceByClientId)
	return
}

func (s *RelayManagementService) validateClient(clientId string) error {
	if !s.clientManager.HasClient(clientId) {
		return errors.New(fmt.Sprintf("invalid client id %s, can not find client id from client manager.", clientId))
	}
	return nil
}

func (s *RelayManagementService) RegisterService(request *service_common.ServiceRequest, pathParams map[string]string, queryParams map[string]string) (err error) {
	defer s.Logger().Printf("service %v registration result: %s",
		utils.ConditionalPick(request != nil, request.Message, nil),
		utils.ConditionalPick(err != nil, err, "success"))

	s.Logger().Println("register service: ", utils.ConditionalPick(request != nil, request.Message, nil))

	if err = s.validateClient(request.From()); err != nil {
		return err
	}
	descriptor, err := server_utils.ParseServiceDescriptor(request.Payload())
	if err != nil {
		return err
	}
	client := s.clientManager.GetClient(descriptor.Provider.Id)
	if client == nil {
		return errors.New("unable to find the client by providerId " + descriptor.Provider.Id)
	}
	service := s.servicePool.Get().(service.IRelayService)
	service.Init(*descriptor, client, client.MessageRelayExecutor())
	err = s.serviceManager.RegisterService(descriptor.Id, service)
	if err != nil {
		return err
	}
	s.ResolveByAck(request)
	return nil
}

func (s *RelayManagementService) UnregisterService(request *service_common.ServiceRequest, pathParams map[string]string, queryParams map[string]string) (err error) {
	defer s.Logger().Printf("service %v un-registration result: %s",
		utils.ConditionalPick(request != nil, request.Message, nil),
		utils.ConditionalPick(err != nil, err, "success"))
	s.Logger().Println("un-register service: ", utils.ConditionalPick(request != nil, request.Message, nil))
	if err = s.validateClient(request.From()); err != nil {
		return err
	}
	descriptor, err := server_utils.ParseServiceDescriptor(request.Payload())
	if err != nil {
		return err
	}
	if descriptor.Provider.Id != request.From() {
		return errors.New(fmt.Sprintf("descriptor provider id(%s) does not match client id(%s)", descriptor.Provider.Id, request.From()))
	}
	service := s.serviceManager.GetService(descriptor.Id)
	if service == nil {
		return servererror.NewNoSuchServiceError(descriptor.Id)
	}
	if service.Provider().Id() != request.From() {
		return errors.New(fmt.Sprintf("actual service provider id(%s) does not match client id(%s)", service.Provider().Id(), request.From()))
	}
	err = s.serviceManager.UnregisterService(descriptor.Id)
	if err != nil {
		return err
	}
	// free service
	s.servicePool.Put(service)
	s.ResolveByAck(request)
	return nil
}

func (s *RelayManagementService) UpdateService(request *service_common.ServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	if err := s.validateClient(request.From()); err != nil {
		return err
	}
	descriptor, err := server_utils.ParseServiceDescriptor(request.Payload())
	if err != nil {
		return err
	}
	if descriptor.Provider.Id != request.From() {
		return errors.New(fmt.Sprintf("descriptor provider id(%s) does not match client id(%s)", descriptor.Provider.Id, request.From()))
	}
	err = s.serviceManager.UpdateService(*descriptor)
	if err != nil {
		// log error
		return err
	}
	s.ResolveByAck(request)
	return nil
}

func (s *RelayManagementService) GetServiceByClientId(request *service_common.ServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	clientId := pathParams[":clientId"]
	if clientId == "" {
		return errors.New("parameter [:clientId] is missing")
	}
	services := s.serviceManager.GetServicesByClientId(clientId)
	marshalled, err := json.Marshal(services)
	if err != nil {
		return err
	}
	request.Resolve(messages.NewMessage(request.Id(), s.HostInfo().Id, request.From(), request.Uri(), messages.MessageTypeServiceResponse, marshalled))
	return nil
}

func (s *RelayManagementService) tryToRestoreDeadServicesFromReconnectedClient(clientId string) (err error) {
	defer s.Logger().Printf("restore service from client %s result: %s", clientId, utils.ConditionalPick(err != nil, err, "success"))
	s.Logger().Println("restore services from client ", clientId)
	s.serviceManager.WithServicesFromClientId(clientId, func(services []service.IService) {
		client := s.clientManager.GetClient(clientId)
		if client == nil {
			err = servererror.NewNoSuchClientError(clientId)
			return
		}
		for i := range services {
			if services[i] != nil {
				if err = services[i].(service.IRelayService).RestoreExternally(client); err != nil {
					return
				}
			}
		}
	})
	return
}
