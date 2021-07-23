package services

import (
	"wsdk/relay_server/context"
	service2 "wsdk/relay_server/controllers/service_manager"
	"wsdk/relay_server/service_base"
	"wsdk/relay_server/services/messaging"
	"wsdk/relay_server/services/pubsub"
	"wsdk/relay_server/services/relay_management"
	"wsdk/relay_server/services/status"
)

var serviceInstances map[string]service_base.INativeService

func init() {
	instantiateInstances()
}

// new services need to be defined here to be registered
// all services should be assigned/newed in order of dependency
func instantiateInstances() {
	serviceInstances = make(map[string]service_base.INativeService)
	serviceInstances[messaging.ID] = new(messaging.MessagingService)
	serviceInstances[pubsub.ID] = new(pubsub.PubSubService)
	serviceInstances[relay_management.ID] = new(relay_management.RelayManagementService)
	serviceInstances[status.ID] = new(status.StatusService)
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

func InitNativeServices(serviceManager service2.IServiceManager) (err error) {
	for k, v := range serviceInstances {
		err = v.Init()
		if err != nil {
			context.Ctx.Logger().Printf("native service_manager %s init failed due to %s", k, err.Error())
			resetInstances()
			return
		}
		err = serviceManager.RegisterService(context.Ctx.Server().Id(), v)
		if err != nil {
			context.Ctx.Logger().Printf("native service_manager %s registration failed due to %s", k, err.Error())
			resetInstances()
			return
		}
	}
	return
}