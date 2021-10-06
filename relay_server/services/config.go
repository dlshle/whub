package services

import (
	"errors"
	"fmt"
	"wsdk/relay_server/config"
	"wsdk/relay_server/context"
	"wsdk/relay_server/module_base"
	"wsdk/relay_server/modules/service_manager"
	"wsdk/relay_server/service_base"
	"wsdk/relay_server/services/auth_service"
	"wsdk/relay_server/services/client_management"
	"wsdk/relay_server/services/messaging"
	"wsdk/relay_server/services/service_management"
	"wsdk/relay_server/services/status"
)

var serviceInstances map[string]service_base.INativeService
var initiated bool

func init() {
	instantiateInstances()
	initiated = false
}

// new services need to be defined here to be registered
// all services should be assigned/newed in order of dependency
func instantiateInstances() {
	serviceInstances = make(map[string]service_base.INativeService)
	serviceInstances[messaging.ID] = new(messaging.MessagingService)
	serviceInstances[service_management.ID] = new(service_management.ServiceManagementService)
	serviceInstances[status.ID] = new(status.StatusService)
	serviceInstances[client_management.ID] = new(client_management.ClientManagementService)
	serviceInstances[auth_service.ID] = new(auth_service.AuthService)
	cleanUpServiceInstances()
}

func cleanUpServiceInstances() {
	disabledServices := config.Config.DisabledServices
	for _, svc := range disabledServices {
		if serviceInstances[svc] != nil {
			delete(serviceInstances, svc)
		}
	}
}

func clearInstances() {
	for k := range serviceInstances {
		delete(serviceInstances, k)
	}
}

func resetInstances() {
	clearInstances()
	instantiateInstances()
}

func InitNativeServices() (err error) {
	if initiated {
		return errors.New("services has already been initiated")
	}
	serviceManager := module_base.Manager.GetModule(service_manager.ID).(service_manager.IServiceManagerModule)
	for k, v := range serviceInstances {
		err = v.Init()
		if err != nil {
			context.Ctx.Logger().Printf("native service %s init failed due to %s", k, err.Error())
			resetInstances()
			return
		}
		err = serviceManager.RegisterService(context.Ctx.Server().Id(), v)
		if err != nil {
			context.Ctx.Logger().Printf("native service %s registration failed due to %s", k, err.Error())
			resetInstances()
			return
		}
	}
	initiated = true
	return
}

// this needs to be called before server starts
func AddNativeService(serviceId string, serviceInstance service_base.INativeService) error {
	if initiated {
		return errors.New("all native services have already initiated")
	}
	if serviceInstances[serviceId] != nil {
		return errors.New(fmt.Sprintf("native service %s already exists", serviceId))
	}
	serviceInstances[serviceId] = serviceInstance
	return nil
}
