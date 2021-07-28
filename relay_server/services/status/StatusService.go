package status

import (
	service_common "wsdk/relay_common/service"
	"wsdk/relay_server/container"
	"wsdk/relay_server/controllers/anonymous_client_manager"
	"wsdk/relay_server/controllers/client_manager"
	"wsdk/relay_server/controllers/service_manager"
	"wsdk/relay_server/controllers/status"
	"wsdk/relay_server/service_base"
)

const (
	ID             = "status"
	RouteGetStatus = "/get" // payload = service descriptor
)

type StatusService struct {
	service_base.INativeService
	systemStatusController status.IServerStatusController                   `$inject:""`
	serviceManager         service_manager.IServiceManager                  `$inject:""`
	clientManager          client_manager.IClientManager                    `$inject:""`
	anonymousClientManager anonymous_client_manager.IAnonymousClientManager `$inject:""`
}

func (s *StatusService) Init() error {
	s.INativeService = service_base.NewNativeService(ID,
		"server status",
		service_common.ServiceTypeInternal,
		service_common.ServiceAccessTypeSocket,
		service_common.ServiceExecutionSync)
	err := container.Container.Fill(s)
	if err != nil {
		return err
	}
	if err = s.initRoutes(); err != nil {
		return err
	}
	return nil
}

func (s *StatusService) initRoutes() error {
	return s.RegisterRoute(RouteGetStatus, s.GetStatus)
}

func (s *StatusService) initPubSubTopic() error {
	// TODO should make a pubsub controller to handle internal pubsub topics more easily
	return nil
}

func (s *StatusService) GetStatus(request *service_common.ServiceRequest, pathParams map[string]string, queryParams map[string]string) (err error) {
	// TODO should check auth scope
	sysStatusJsonByte, err := s.systemStatusController.GetServerStat().JsonByte()
	if err != nil {
		return err
	}
	s.ResolveByResponse(request, sysStatusJsonByte)
	return nil
}
