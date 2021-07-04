package events

import (
	"wsdk/relay_common/messages"
	"wsdk/relay_common/notification"
	"wsdk/relay_server/context"
)

const (
	EventClientDisconnected      = "EClientDisconnected"
	EventClientUnexpectedClosure = "EClientUnexpectedClosure"
	EventClientConnected         = "EClientReconnected"
	EventServiceRegistered       = "EServiceRegistered"
	EventServiceUnregistered     = "EServiceUnregistered"

	EventServerStarted = "EServerStarted"
	EventServerClosed  = "EServerClosed"
	EventServerError   = "EServerDown"
)

func EmitEvent(eventId string, message string) {
	context.Ctx.NotificationEmitter().Notify(eventId, messages.NewNotification(eventId, message))
}

func OnEvent(eventId string, listener notification.MessageListener) (notification.Disposable, error) {
	return context.Ctx.NotificationEmitter().On(eventId, listener)
}
