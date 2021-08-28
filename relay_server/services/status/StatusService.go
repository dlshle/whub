package status

import (
	"encoding/json"
	service_common "wsdk/relay_common/service"
	"wsdk/relay_server/container"
	"wsdk/relay_server/core/service_manager"
	"wsdk/relay_server/core/status"
	"wsdk/relay_server/service_base"
)

const (
	ID               = "status"
	RouteGetStatus   = "/get"      // payload = service descriptor
	RouteInfo        = "/info"     // payload = server  descriptor
	RouteGetServices = "/services" // payload = all internal services
)

type StatusService struct {
	service_base.INativeService
	systemStatusController status.IServerStatusController  `$inject:""`
	serviceManager         service_manager.IServiceManager `$inject:""`
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
	routeMap := make(map[string]service_common.RequestHandler)
	routeMap[RouteGetStatus] = s.GetStatus
	routeMap[RouteGetStatus] = s.GetAllInternalServices
	routeMap[RouteInfo] = s.GetInfo
	return s.InitRoutes(routeMap)
}

func (s *StatusService) initPubSubTopic() error {
	// TODO should make a pubsub controller to handle internal pubsub topics more easily
	return nil
}

func (s *StatusService) GetStatus(request service_common.IServiceRequest, pathParams map[string]string, queryParams map[string]string) (err error) {
	if request.From() == "" {
		return s.ResolveByInvalidCredential(request)
	}
	sysStatusJsonByte, err := s.systemStatusController.GetServerStat().JsonByte()
	if err != nil {
		return err
	}
	s.ResolveByResponse(request, sysStatusJsonByte)
	return nil
}

func (s *StatusService) GetAllInternalServices(request service_common.IServiceRequest, pathParams map[string]string, queryParams map[string]string) (err error) {
	if request.From() == "" {
		return s.ResolveByInvalidCredential(request)
	}
	servicesJsonByte, err := json.Marshal(s.serviceManager.DescribeAllServices())
	if err != nil {
		return err
	}
	s.ResolveByResponse(request, servicesJsonByte)
	return nil
}

func (s *StatusService) GetInfo(request service_common.IServiceRequest, pathParams map[string]string, queryParams map[string]string) (err error) {
	s.ResolveByResponse(request, ([]byte)(s.HostInfo().String()))
	return nil
}
