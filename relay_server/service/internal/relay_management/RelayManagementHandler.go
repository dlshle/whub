package relay_management

import (
	"errors"
	"wsdk/relay_common"
	service_common "wsdk/relay_common/service"
	"wsdk/relay_server"
	"wsdk/relay_server/service"
)

type RelayManagementHandler struct {
	ctx            *relay_common.WRContext
	serviceManager service.IServiceManager
	clientManager  relay_server.IClientManager
}

func (h *RelayManagementHandler) RegisterService(request *service_common.ServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	descriptor, err := relay_server.ParseServiceDescriptor(request.Payload())
	if err != nil {
		return err
	}
	client := h.clientManager.GetClient(descriptor.Provider.Id)
	if client == nil {
		return errors.New("unable to find the client by providerId " + descriptor.Provider.Id)
	}
	service := service.NewRelayService(h.ctx, *descriptor, client, client.MessageRelayExecutor())
	return h.serviceManager.RegisterService(descriptor.Id, service)
}

// TODO unregister service, etc...
