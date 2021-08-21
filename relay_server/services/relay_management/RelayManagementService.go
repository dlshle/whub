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
	"wsdk/relay_server/core/client_manager"
	"wsdk/relay_server/core/connection_manager"
	"wsdk/relay_server/core/service_manager"
	servererror "wsdk/relay_server/errors"
	"wsdk/relay_server/events"
	request_executor "wsdk/relay_server/request"
	"wsdk/relay_server/service_base"
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
	*service_base.NativeService
	clientManager     client_manager.IClientManager         `$inject:""`
	serviceManager    service_manager.IServiceManager       `$inject:""`
	connectionManager connection_manager.IConnectionManager `$inject:""`
	servicePool       *sync.Pool
}

func (s *RelayManagementService) Init() error {
	s.NativeService = service_base.NewNativeService(ID, "relay management service", service_common.ServiceTypeInternal, service_common.ServiceAccessTypeSocket, service_common.ServiceExecutionSync)
	s.servicePool = &sync.Pool{
		New: func() interface{} {
			return new(service_base.RelayService)
		},
	}
	err := container.Container.Fill(s)
	if err != nil {
		return err
	}
	return s.init()
}

func (s *RelayManagementService) init() error {
	s.initNotificationHandlers()
	return s.initRoutes()
}

func (s *RelayManagementService) initNotificationHandlers() {
	events.OnEvent(events.EventClientConnectionGone, func(message messages.IMessage) {
		clientId := string(message.Payload()[:])
		s.serviceManager.UnregisterAllServicesFromClientId(clientId)
	})
	events.OnEvent(events.EventClientUnexpectedClosure, func(message messages.IMessage) {
		clientId := string(message.Payload()[:])
		s.serviceManager.WithServicesFromClientId(clientId, func(services []service_base.IService) {
			for _, svc := range services {
				svc.Kill()
			}
		})
	})
	events.OnEvent(events.EventServiceNewProvider, func(message messages.IMessage) {
		clientId := string(message.Payload()[:])
		s.tryToRestoreDeadServicesFromReconnectedClient(clientId)
	})
}

func (s *RelayManagementService) initRoutes() error {
	routeMap := make(map[string]service_common.RequestHandler)
	routeMap[RouteRegisterService] = s.RegisterService
	routeMap[RouteUnregisterService] = s.UnregisterService
	routeMap[RouteUpdateService] = s.UpdateService
	routeMap[RouteGetAllServices] = s.GetAllRelayServices
	routeMap[RouteGetServicesByClientId] = s.GetServiceByClientId
	return s.InitRoutes(routeMap)
}

func (s *RelayManagementService) validateClientConnection(request service_common.IServiceRequest) error {
	if request.GetContext(connection_manager.IsSyncConnContextKey).(bool) {
		return errors.New("connection type not supported")
	}
	if request.From() == "" {
		return errors.New("invalid credential")
	}
	return nil
}

func (s *RelayManagementService) RegisterService(request service_common.IServiceRequest, pathParams map[string]string, queryParams map[string]string) (err error) {
	defer func() {
		if err != nil {
			s.Logger().Printf("service registration from %s failed due to %s", request.From(), err.Error())
		} else {
			s.Logger().Println("service registration from %s succeeded", request.From())
		}
	}()
	s.Logger().Println("register service: ", utils.ConditionalPick(request != nil, request.Message(), nil))
	if err = s.validateClientConnection(request); err != nil {
		return err
	}
	descriptor, err := server_utils.ParseServiceDescriptor(request.Payload())
	if err != nil {
		return err
	}
	client, err := s.clientManager.GetClientWithErrOnNotFound(descriptor.Provider.Id)
	if err != nil {
		return err
	}
	if s.serviceManager.HasService(descriptor.Id) {
		// service already running, notify service executor to add extra connection
		events.EmitEvent(events.EventServiceNewProvider, descriptor.Id)
		s.ResolveByAck(request)
		return nil
	}
	service := s.servicePool.Get().(service_base.IRelayService)
	service.Init(descriptor, client, request_executor.NewRelayServiceRequestExecutor(descriptor.Id, client.Id()))
	err = s.serviceManager.RegisterService(descriptor.Id, service)
	if err != nil {
		s.servicePool.Put(service)
		return err
	}
	s.ResolveByAck(request)
	return nil
}

func (s *RelayManagementService) UnregisterService(request service_common.IServiceRequest, pathParams map[string]string, queryParams map[string]string) (err error) {
	defer s.Logger().Printf("service %v un-registration result: %s",
		utils.ConditionalPick(request != nil, request.Message(), nil),
		utils.ConditionalPick(err != nil, err, "success"))
	s.Logger().Println("un-register service: ", utils.ConditionalPick(request != nil, request.Message(), nil))
	if err = s.validateClientConnection(request); err != nil {
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

func (s *RelayManagementService) UpdateService(request service_common.IServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	if err := s.validateClientConnection(request); err != nil {
		return err
	}
	descriptor, err := server_utils.ParseServiceDescriptor(request.Payload())
	if err != nil {
		return err
	}
	if descriptor.Provider.Id != request.From() {
		return errors.New(fmt.Sprintf("descriptor provider id(%s) does not match client id(%s)", descriptor.Provider.Id, request.From()))
	}
	err = s.serviceManager.UpdateService(descriptor)
	if err != nil {
		// log error
		return err
	}
	s.ResolveByAck(request)
	return nil
}

func (s *RelayManagementService) GetServiceByClientId(request service_common.IServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	clientId := pathParams["clientId"]
	if clientId == "" {
		return errors.New("parameter [:clientId] is missing")
	}
	_, err := s.clientManager.GetClientWithErrOnNotFound(clientId)
	if err != nil {
		return err
	}
	services := s.serviceManager.GetServicesByClientId(clientId)
	descriptors := make([]service_common.ServiceDescriptor, len(services), len(services))
	for i, svc := range services {
		descriptors[i] = svc.Describe()
	}
	marshalled, err := json.Marshal(descriptors)
	if err != nil {
		return err
	}
	request.Resolve(messages.NewMessage(request.Id(), s.HostInfo().Id, request.From(), request.Uri(), messages.MessageTypeServiceResponse, marshalled))
	return nil
}

func (s *RelayManagementService) GetAllRelayServices(request service_common.IServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	services := s.serviceManager.DescribeAllRelayServices()
	marshalled, err := json.Marshal(services)
	if err != nil {
		return err
	}
	s.ResolveByResponse(request, marshalled)
	return nil
}

func (s *RelayManagementService) tryToRestoreDeadServicesFromReconnectedClient(clientId string) (err error) {
	defer s.Logger().Printf("restore service from client %s result: %s", clientId, utils.ConditionalPick(err != nil, err, "success"))
	s.Logger().Println("restore services from client ", clientId)
	s.serviceManager.WithServicesFromClientId(clientId, func(services []service_base.IService) {
		client, cerr := s.clientManager.GetClientWithErrOnNotFound(clientId)
		if cerr != nil {
			err = cerr
			return
		}
		for i := range services {
			if services[i] != nil {
				if err = services[i].(service_base.IRelayService).RestoreExternally(client); err != nil {
					return
				}
			}
		}
	})
	return
}
