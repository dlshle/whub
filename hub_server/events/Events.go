package events

import (
	"whub/hub_common/messages"
	"whub/hub_common/notification"
	"whub/hub_server/context"
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

func OffEvent(eventId string, listener notification.MessageListener) {
	context.Ctx.NotificationEmitter().Off(eventId, listener)
}
