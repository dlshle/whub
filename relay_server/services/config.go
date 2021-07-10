package services

import (
	"errors"
	"wsdk/relay_server/container"
	"wsdk/relay_server/context"
	"wsdk/relay_server/managers"
	"wsdk/relay_server/service"
	"wsdk/relay_server/services/messaging"
	"wsdk/relay_server/services/pubsub"
	"wsdk/relay_server/services/relay_management"
)

var serviceInstances map[string]service.INativeService

func init() {
	serviceInstances = make(map[string]service.INativeService)
	serviceInstances[messaging.ID] = new(messaging.MessagingService)
	serviceInstances[pubsub.ID] = new(pubsub.PubSubService)
	serviceInstances[relay_management.ID] = new(relay_management.RelayManagementService)
}

func InitNativeServices() error {
	serviceManager := container.Container.GetById(managers.ServiceManagerId).(managers.IServiceManager)
	if serviceManager == nil {
		return errors.New("unable to get serviceManager")
	}
	for k, v := range serviceInstances {
		err := v.Init()
		if err != nil {
			context.Ctx.Logger().Printf("native service %s init failed due to %s", k, err.Error())
		}
		serviceManager.RegisterService(context.Ctx.Server().Id(), v)
	}
	return nil
}
