package relay_management

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"wsdk/common/utils"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/roles"
	service_common "wsdk/relay_common/service"
	"wsdk/relay_server/client"
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
	ID                                 = "relay"
	RouteRegisterService               = "/register"   // payload = service descriptor
	RouteUnregisterService             = "/unregister" // payload = service descriptor
	RouteUpdateService                 = "/update"     // payload = service descriptor
	RouteGetAllServices                = "/services"   // need privilege, respond with all relayed services
	RouteGetServicesByClientId         = "/clients/:clientId"
	RouteUpdateProviderConnection      = "/providers" // need privilege, need to check if client has service
	RouteGetServiceProviderConnections = "/services/:id/providers"
)

type RelayManagementService struct {
	*service_base.NativeService
	clientManager  client_manager.IClientManager   `$inject:""`
	serviceManager service_manager.IServiceManager `$inject:""`
	servicePool    *sync.Pool
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
	return s.InitHandlers(service_common.NewRequestHandlerMapBuilder().
		Post(RouteRegisterService, s.RegisterService).
		Delete(RouteUnregisterService, s.UnregisterService).
		Put(RouteUpdateService, s.UpdateService).
		Get(RouteGetAllServices, s.GetAllRelayServices).
		Get(RouteGetServicesByClientId, s.GetServiceByClientId).
		Patch(RouteUpdateProviderConnection, s.UpdateServiceProviderConnection).
		Get(RouteGetServiceProviderConnections, s.GetServiceProviderConnections).Build())
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
	service := s.createRelayService(client, descriptor)
	err = s.serviceManager.RegisterService(descriptor.Id, service)
	if err != nil {
		s.servicePool.Put(service)
		return err
	}
	s.ResolveByAck(request)
	return nil
}

func (s *RelayManagementService) createRelayService(provider *client.Client, descriptor service_common.ServiceDescriptor) service_base.IService {
	service := s.servicePool.Get().(service_base.IRelayService)
	service.Init(descriptor, provider, request_executor.NewRelayServiceRequestExecutor(descriptor.Id, provider.Id()))
	return service
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
	s.ResolveByResponse(request, marshalled)
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

func (s *RelayManagementService) UpdateServiceProviderConnection(request service_common.IServiceRequest, pathParams map[string]string, queryParams map[string]string) (err error) {
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

func (s *RelayManagementService) GetServiceProviderConnections(request service_common.IServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	if request.From() == "" {
		return s.ResolveByInvalidCredential(request)
	}
	serviceId := pathParams["id"]
	if serviceId == "" {
		return s.ResolveByError(request, messages.MessageTypeSvcBadRequestError, "invalid service id")
	}
	svc := s.serviceManager.GetService(serviceId)
	if svc == nil {
		return s.ResolveByError(request, messages.MessageTypeSvcNotFoundError, fmt.Sprintf("can not find service by id %s", serviceId))
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

func (s *RelayManagementService) assembleConnJsonArr(conns []connection.IConnection) string {
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
