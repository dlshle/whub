package relay_management

import (
	"errors"
	"wsdk/relay_common/messages"
	service_common "wsdk/relay_common/service"
	"wsdk/relay_server"
	"wsdk/relay_server/service"
)

const (
	ID                     = "relay"
	RouteRegisterService   = "/register"   // payload = service descriptor
	RouteUnregisterService = "/unregister" // payload = service descriptor
	RouteUpdateService     = "/update"     // payload = service descriptor
	RouteGetAllServices    = "/services"   // need privilege, respond with all relayed services
)

type RelayManagementService struct {
	*service.InternalService
	clientManager  relay_server.IClientManager
	serviceManager service.IServiceManager
}

func New(ctx *relay_server.Context, serviceManager service.IServiceManager, clientManager relay_server.IClientManager) service.IServerService {
	relayManagementService := &RelayManagementService{
		service.NewInternalService(ctx, ID, "relay management service", service_common.ServiceTypeInternal, service_common.ServiceAccessTypeSocket, service_common.ServiceExecutionSync),
		clientManager,
		serviceManager,
	}
	relayManagementService.init()
	return relayManagementService
}

func (s *RelayManagementService) init() {
	s.initNotificationHandlers()
	s.initRoutes()
}

func (s *RelayManagementService) initNotificationHandlers() {
	s.Ctx().NotificationEmitter().On(relay_server.EventClientDisconnected, func(message *messages.Message) {
		clientId := string(message.Payload()[:])
		s.serviceManager.UnregisterAllServicesFromClientId(clientId)
	})
	s.Ctx().NotificationEmitter().On(relay_server.EventClientUnexpectedClosure, func(message *messages.Message) {
		clientId := string(message.Payload()[:])
		s.serviceManager.WithServicesFromClientId(clientId, func(services []service.IServerService) {
			for _, svc := range services {
				svc.Kill()
			}
		})
	})
	s.Ctx().NotificationEmitter().On(relay_server.EventClientConnected, func(message *messages.Message) {
		clientId := string(message.Payload()[:])
		s.tryToRestoreDeadServicesFromReconnectedClient(clientId)
	})
}

func (s *RelayManagementService) initRoutes() {
	s.RegisterRoute(RouteRegisterService, s.RegisterService)
	s.RegisterRoute(RouteUnregisterService, s.UnregisterService)
}

func (s *RelayManagementService) RegisterService(request *service_common.ServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	descriptor, err := relay_server.ParseServiceDescriptor(request.Payload())
	if err != nil {
		return err
	}
	client := s.clientManager.GetClient(descriptor.Provider.Id)
	if client == nil {
		return errors.New("unable to find the client by providerId " + descriptor.Provider.Id)
	}
	service := service.NewRelayService(s.Ctx(), *descriptor, client, client.MessageRelayExecutor())
	return s.serviceManager.RegisterService(descriptor.Id, service)
}

func (s *RelayManagementService) UnregisterService(request *service_common.ServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	descriptor, err := relay_server.ParseServiceDescriptor(request.Payload())
	if err != nil {
		return err
	}
	return s.serviceManager.UnregisterService(descriptor.Id)
}

func (s *RelayManagementService) tryToRestoreDeadServicesFromReconnectedClient(clientId string) (err error) {
	// TODO need log
	s.serviceManager.WithServicesFromClientId(clientId, func(services []service.IServerService) {
		client := s.clientManager.GetClient(clientId)
		if client == nil {
			err = relay_server.NewNoSuchClientError(clientId)
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
