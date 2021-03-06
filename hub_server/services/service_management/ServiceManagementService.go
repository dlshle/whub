package service_management

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"whub/common/utils"
	"whub/hub_common/connection"
	"whub/hub_common/messages"
	"whub/hub_common/roles"
	service_common "whub/hub_common/service"
	"whub/hub_server/client"
	servererror "whub/hub_server/errors"
	"whub/hub_server/events"
	"whub/hub_server/module_base"
	"whub/hub_server/modules/client_manager"
	"whub/hub_server/modules/connection_manager"
	"whub/hub_server/modules/service_manager"
	request_executor "whub/hub_server/request"
	"whub/hub_server/service_base"
	server_utils "whub/hub_server/utils"
)

const (
	ID                                 = "services"
	RouteRegisterService               = "/register"   // payload = service descriptor
	RouteUnregisterService             = "/unregister" // payload = service descriptor
	RouteUpdateService                 = "/update"     // payload = service descriptor
	RouteGetAllServices                = "/services"   // need privilege, respond with all relayed services
	RouteGetServicesByClientId         = "/clients/:clientId"
	RouteUpdateProviderConnection      = "/providers" // need privilege, need to check if client has service
	RouteGetServiceProviderConnections = "/:id/providers"
	RouteGetServiceById                = "/:id"
)

type ServiceManagementService struct {
	*service_base.NativeService
	clientManager  client_manager.IClientManagerModule   `module:""`
	serviceManager service_manager.IServiceManagerModule `module:""`
	servicePool    *sync.Pool
}

func (s *ServiceManagementService) Init() error {
	s.NativeService = service_base.NewNativeService(ID, "relay management service", service_common.ServiceTypeInternal, service_common.ServiceAccessTypeSocket, service_common.ServiceExecutionSync)
	s.servicePool = &sync.Pool{
		New: func() interface{} {
			return new(service_base.RelayService)
		},
	}
	err := module_base.Manager.AutoFill(s)
	if err != nil {
		return err
	}
	return s.init()
}

func (s *ServiceManagementService) init() error {
	s.initNotificationHandlers()
	return s.initRoutes()
}

func (s *ServiceManagementService) initNotificationHandlers() {
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

func (s *ServiceManagementService) initRoutes() error {
	return s.RegisterRoutes(service_common.NewRequestHandlerMapBuilder().
		Get(RouteGetServiceById, s.GetServiceById).
		Post(RouteRegisterService, s.RegisterService).
		Delete(RouteUnregisterService, s.UnregisterService).
		Put(RouteUpdateService, s.UpdateService).
		Get(RouteGetAllServices, s.GetAllRelayServices).
		Get(RouteGetServicesByClientId, s.GetServiceByClientId).
		Patch(RouteUpdateProviderConnection, s.UpdateServiceProviderConnection).
		Get(RouteGetServiceProviderConnections, s.GetServiceProviderConnections).Build())
}

func (s *ServiceManagementService) validateClientConnection(request service_common.IServiceRequest) error {
	if request.GetContext(connection_manager.IsSyncConnContextKey).(bool) {
		return errors.New("connection type not supported")
	}
	if request.From() == "" {
		return errors.New("invalid credential")
	}
	return nil
}

func (s *ServiceManagementService) GetServiceById(request service_common.IServiceRequest, pathParams map[string]string, queryParams map[string]string) (err error) {
	id := pathParams["id"]
	if id == "" {
		return s.ResolveByError(request, messages.MessageTypeSvcBadRequestError, "invalid id path param")
	}
	svc := s.serviceManager.GetService(id)
	if svc == nil {
		return s.ResolveByError(request, messages.MessageTypeSvcNotFoundError, fmt.Sprintf("can not find service by id %s", id))
	}
	marshalled, err := json.Marshal(svc.Describe())
	if err != nil {
		return err
	}
	return s.ResolveByResponse(request, marshalled)
}

func (s *ServiceManagementService) RegisterService(request service_common.IServiceRequest, pathParams map[string]string, queryParams map[string]string) (err error) {
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
	service := s.createRelayService(client, descriptor)
	err = s.serviceManager.RegisterService(descriptor.Id, service)
	if err != nil {
		s.servicePool.Put(service)
		return err
	}
	s.ResolveByAck(request)
	return nil
}

func (s *ServiceManagementService) createRelayService(provider *client.Client, descriptor service_common.ServiceDescriptor) service_base.IService {
	service := s.servicePool.Get().(service_base.IRelayService)
	service.Init(descriptor, provider, request_executor.NewRelayServiceRequestExecutor(descriptor.Id, provider.Id()))
	return service
}

func (s *ServiceManagementService) UnregisterService(request service_common.IServiceRequest, pathParams map[string]string, queryParams map[string]string) (err error) {
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

func (s *ServiceManagementService) UpdateService(request service_common.IServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	if err := s.validateClientConnection(request); err != nil {
		return s.ResolveByInvalidCredential(request)
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

func (s *ServiceManagementService) GetServiceByClientId(request service_common.IServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
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
	s.ResolveByResponse(request, marshalled)
	return nil
}

func (s *ServiceManagementService) GetAllRelayServices(request service_common.IServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	services := s.serviceManager.DescribeAllRelayServices()
	marshalled, err := json.Marshal(services)
	if err != nil {
		return err
	}
	s.ResolveByResponse(request, marshalled)
	return nil
}

func (s *ServiceManagementService) UpdateServiceProviderConnection(request service_common.IServiceRequest, pathParams map[string]string, queryParams map[string]string) (err error) {
	err = s.validateClientConnection(request)
	if err != nil {
		return s.ResolveByInvalidCredential(request)
	}
	addr := request.GetContext(connection_manager.AddrContextKey)
	if addr == nil || addr == "" {
		return errors.New("invalid provider address")
	}
	descriptor, err := server_utils.ParseServiceDescriptor(request.Payload())
	if err != nil {
		return err
	}
	service := s.serviceManager.GetService(descriptor.Id)
	if service == nil {
		return s.ResolveByError(request, messages.MessageTypeSvcNotFoundError, fmt.Sprintf("can not find service by id %s", descriptor.Id))
	}
	if service.Provider().Id() != request.From() {
		return s.ResolveByError(request, messages.MessageTypeSvcForbiddenError, fmt.Sprintf("client %s is not the provider for service %s", request.From(), service.Provider().Id()))
	}
	err = service.(service_base.IRelayService).UpdateProviderConnection(addr.(string))
	if err != nil {
		return err
	}
	return s.ResolveByResponse(request, ([]byte)(descriptor.String()))
}

func (s *ServiceManagementService) tryToRestoreDeadServicesFromReconnectedClient(clientId string) (err error) {
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

func (s *ServiceManagementService) GetServiceProviderConnections(request service_common.IServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	if request.From() == "" {
		return s.ResolveByInvalidCredential(request)
	}
	serviceId := pathParams["id"]
	if serviceId == "" {
		return s.ResolveByError(request, messages.MessageTypeSvcBadRequestError, "invalid service id")
	}
	svc := s.serviceManager.GetService(serviceId)
	if svc == nil {
		return s.ResolveByError(request, messages.MessageTypeSvcNotFoundError, fmt.Sprintf("can not find service by id [%s]", serviceId))
	}
	if svc.ServiceType() != service_common.ServiceTypeProxy {
		return s.ResolveByError(request, messages.MessageTypeSvcBadRequestError, fmt.Sprintf("service [%s] is not a relay service", svc.Id()))
	}
	me, err := s.clientManager.GetClient(request.From())
	if err != nil {
		return err
	}
	if svc.Provider().Id() != request.From() && me.CType() < roles.ClientTypeManager {
		return s.ResolveByInvalidCredential(request)
	}
	conns := svc.(service_base.IRelayService).GetProviderConnections()
	return s.ResolveByResponse(request, ([]byte)(s.assembleConnJsonArr(conns)))
}

func (s *ServiceManagementService) assembleConnJsonArr(conns []connection.IConnection) string {
	var builder strings.Builder
	builder.WriteByte('[')
	for i, conn := range conns {
		if i == len(conns)-1 {
			builder.WriteString(fmt.Sprintf("%s", conn.String()))
		} else {
			builder.WriteString(fmt.Sprintf("%s,", conn.String()))
		}
	}
	builder.WriteByte(']')
	return builder.String()
}
