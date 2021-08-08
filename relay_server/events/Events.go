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
	EventServiceUpdated          = "EServiceUpdated"
	EventServiceUnregistered     = "EServiceUnregistered"
	EventServiceNewProvider      = "EServiceNewProvider"

	EventServerStarted = "EServerStarted"
	EventServerClosed  = "EServerClosed"
	EventServerError   = "EServerDown"

	EventTopicRemoval = "ETopicRemoval"

	EventClientConnectionEstablished = "EClientConnectionEstablished"
	EventClientConnectionClosed      = "EClientConnectionClosed" // one connection closed
	EventClientConnectionGone        = "EClientConnectionGone"   // all connections closed
)

func EmitEvent(eventId string, message string) {
	context.Ctx.NotificationEmitter().Notify(eventId, messages.NewNotification(eventId, message))
}

func OnEvent(eventId string, listener notification.MessageListener) (notification.Disposable, error) {
	return context.Ctx.NotificationEmitter().On(eventId, listener)
}
