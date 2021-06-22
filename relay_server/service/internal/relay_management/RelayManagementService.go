package relay_management

import (
	"wsdk/relay_common"
	service_common "wsdk/relay_common/service"
	"wsdk/relay_server"
	"wsdk/relay_server/service"
)

const (
	RouteRegisterService   = "/register"   // payload = service descriptor
	RouteUnregisterService = "/unregister" // payload = service descriptor
	RouteUpdateService     = "/update"     // payload = service descriptor
)

type RelayManagementService struct {
	*service.InternalService
	handler *RelayManagementHandler
}

func New(ctx *relay_common.WRContext, serviceManager service.IServiceManager, clientManager relay_server.IClientManager) service.IServerService {
	relayManagementService := &RelayManagementService{
		service.NewInternalService(ctx, "relay", "relay management service", service_common.ServiceTypeInternal, service_common.ServiceAccessTypeSocket, service_common.ServiceExecutionSync),
		&RelayManagementHandler{ctx: ctx, serviceManager: serviceManager, clientManager: clientManager},
	}
	relayManagementService.init()
	return relayManagementService
}

func (s *RelayManagementService) init() {
	s.RegisterRoute(RouteRegisterService, s.handler.RegisterService)
}
