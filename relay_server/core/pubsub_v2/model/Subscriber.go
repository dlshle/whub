package model

import "wsdk/relay_server/core/pubsub_v2"

// SubscriberGroup a group of msg receivers that receives messages on one topic, each message is sent to one connection in
// the group. All connection groups in the topic will receive one instance of the message from queue.

type IPubSubConsumer interface {
	Consume(pubsub_v2.IPubSubMessage) error
	Mode() SubscribeMode
}

type SubscribeMode uint8

const (
	SubModePull = 0
	SubModePush = 1
)

func (m SubscribeMode) String() string {
	switch m {
	case SubModePull:
		return "Pull"
	case SubModePush:
		return "Push"
	}
	return "unknown"
}

type ISubscriberGroup interface {
	Mode() SubscribeMode
	LastMsgIndex() uint32
}

type SubscriberGroup struct {
	name         string
	consumers    []IPubSubConsumer
	lastMsgIndex uint32
}
