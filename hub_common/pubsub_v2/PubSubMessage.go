package pubsub_v2

type IPubSubMessage interface {
	TopicId() string
	Index() uint32
	Priority() uint8
	Payload() []byte
}

type PubSubMessage struct {
	topicId  string
	index    uint32
	priority uint8
	payload  []byte
}

func (m *PubSubMessage) TopicId() string {
	return m.topicId
}

func (m *PubSubMessage) Index() uint32 {
	return m.index
}

func (m *PubSubMessage) Priority() uint8 {
	return m.priority
}

func (m *PubSubMessage) Payload() []byte {
	return m.payload
}
