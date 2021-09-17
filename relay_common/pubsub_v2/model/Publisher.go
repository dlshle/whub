package model

import (
	"wsdk/relay_common/pubsub_v2"
)

// Publisher a client that publishes messages

type IPubSubProducer interface {
	Produce(pubsub_v2.IPubSubMessage) error
}

type Publisher struct {
	clientId     string
	lastMsgIndex uint32
}
