package relay_management

import (
	"encoding/json"
	"errors"
	"fmt"
	"wsdk/relay_common/messages"
	service_common "wsdk/relay_common/service"
	"wsdk/relay_server"
	"wsdk/relay_server/container"
	servererror "wsdk/relay_server/errors"
	"wsdk/relay_server/events"
	"wsdk/relay_server/managers"
	"wsdk/relay_server/service"
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
	clientManager  managers.IClientManager
	serviceManager managers.IServiceManager
}

func New() service.IService {
	relayManagementService := &RelayManagementService{
		NativeService:  service.NewNativeService(ID, "relay management service", service_common.ServiceTypeInternal, service_common.ServiceAccessTypeSocket, service_common.ServiceExecutionSync),
		clientManager:  container.Container.GetById(managers.ClientManagerId).(managers.IClientManager),
		serviceManager: container.Container.GetById(managers.ServiceManagerId).(managers.IServiceManager),
	}
	relayManagementService.init()
	return relayManagementService
}

func (s *RelayManagementService) init() {
	s.initNotificationHandlers()
	s.initRoutes()
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

func (s *RelayManagementService) initRoutes() {
	s.RegisterRoute(RouteRegisterService, s.RegisterService)
	s.RegisterRoute(RouteUnregisterService, s.UnregisterService)
	s.RegisterRoute(RouteUpdateService, s.UpdateService)
	s.RegisterRoute(RouteGetServicesByClientId, s.GetServiceByClientId)
}

func (s *RelayManagementService) validateClient(clientId string) error {
	if !s.clientManager.HasClient(clientId) {
		return errors.New(fmt.Sprintf("invalid client id %s, can not find client id from client manager.", clientId))
	}
	return nil
}

func (s *RelayManagementService) RegisterService(request *service_common.ServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	if err := s.validateClient(request.From()); err != nil {
		return err
	}
	descriptor, err := relay_server.ParseServiceDescriptor(request.Payload())
	if err != nil {
		return err
	}
	client := s.clientManager.GetClient(descriptor.Provider.Id)
	if client == nil {
		return errors.New("unable to find the client by providerId " + descriptor.Provider.Id)
	}
	service := service.NewRelayService(*descriptor, client, client.MessageRelayExecutor())
	err = s.serviceManager.RegisterService(descriptor.Id, service)
	if err != nil {
		return err
	}
	return s.ResolveByAck(request)
}

func (s *RelayManagementService) UnregisterService(request *service_common.ServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	if err := s.validateClient(request.From()); err != nil {
		return err
	}
	descriptor, err := relay_server.ParseServiceDescriptor(request.Payload())
	if err != nil {
		return err
	}
	if descriptor.Provider.Id != request.From() {
		return errors.New(fmt.Sprintf("descriptor provider id(%s) does not match client id(%s)", descriptor.Provider.Id, request.From()))
	}
	err = s.serviceManager.UnregisterService(descriptor.Id)
	if err != nil {
		return err
	}
	return s.ResolveByAck(request)
}

func (s *RelayManagementService) UpdateService(request *service_common.ServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	if err := s.validateClient(request.From()); err != nil {
		return err
	}
	descriptor, err := relay_server.ParseServiceDescriptor(request.Payload())
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
	return s.ResolveByAck(request)
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
	return request.Resolve(messages.NewMessage(request.Id(), s.HostInfo().Id, request.From(), request.Uri(), messages.MessageTypeServiceResponse, marshalled))
}

func (s *RelayManagementService) tryToRestoreDeadServicesFromReconnectedClient(clientId string) (err error) {
	// TODO need log
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
