package topic

import (
	"errors"
	"fmt"
	"sync"
	"wsdk/relay_common/messages"
	"wsdk/relay_server/container"
	"wsdk/relay_server/controllers/status"
	"wsdk/relay_server/events"
)

type TopicManager struct {
	topics    map[string]*Topic
	topicPool *sync.Pool
	lock      *sync.RWMutex
}

func (m *TopicManager) initNotificationHandlers() {
	events.OnEvent(events.EventClientDisconnected, func(e *messages.Message) {
		clientId := string(e.Payload()[:])
		for _, t := range m.topics {
			t.CheckAndRemoveSubscriber(clientId)
		}
	})
}

func (m *TopicManager) withWrite(cb func()) {
	m.lock.Lock()
	defer m.lock.Unlock()
	cb()
}

func (m *TopicManager) getTopic(id string) *Topic {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.topics[id]
}

func (m *TopicManager) GetTopic(id string) Topic {
	return *m.getTopic(id)
}

func (m *TopicManager) HasTopic(id string) bool {
	return m.getTopic(id) != nil
}

func (m *TopicManager) SubscribeClientToTopic(clientId string, topicId string) error {
	topic := m.getTopic(topicId)
	if topic == nil {
		return errors.New(fmt.Sprintf("topic %s does not exist", topicId))
	}
	return topic.CheckAndAddSubscriber(clientId)
}

func (m *TopicManager) UnSubscribeClientToTopic(clientId string, topicId string) error {
	topic := m.getTopic(topicId)
	if topic == nil {
		return errors.New(fmt.Sprintf("topic %s does not exist", topicId))
	}
	return topic.CheckAndRemoveSubscriber(clientId)
}

func (m *TopicManager) GetSubscriberIds(topicId string) ([]string, error) {
	topic := m.getTopic(topicId)
	if topic == nil {
		return nil, errors.New(fmt.Sprintf("topic %s does not exist", topicId))
	}
	return topic.Subscribers(), nil
}

func (m *TopicManager) getAllTopics() []*Topic {
	m.lock.RLock()
	defer m.lock.RUnlock()
	topics := make([]*Topic, 0, len(m.topics))
	for _, t := range m.topics {
		topics = append(topics, t)
	}
	return topics
}

func (m *TopicManager) GetAllDescribedTopics() []TopicDescriptor {
	topics := m.getAllTopics()
	topicDescriptors := make([]TopicDescriptor, 0, len(topics))
	for i := range topics {
		topicDescriptors[i] = topics[i].Describe()
	}
	return topicDescriptors
}

func (m *TopicManager) CreateTopic(topicId string, creatorClientId string) Topic {
	topic := m.topicPool.Get().(*Topic)
	topic.Init(topicId, creatorClientId)
	m.withWrite(func() {
		m.topics[topicId] = topic
	})
	return *topic
}

// TODO caller needs to get the subscribers first to notify them and then call this function
func (m *TopicManager) RemoveTopic(topicId string, requestClientId string) error {
	topic := m.getTopic(topicId)
	if topic == nil {
		return errors.New(fmt.Sprintf("topic %s does not exist", topicId))
	}
	if topic.Creator() != requestClientId {
		return errors.New(fmt.Sprintf("can not remove the topic due to [client %s is not the creator of %s]",
			requestClientId,
			topic.Id()))
	}
	m.withWrite(func() {
		delete(m.topics, topic.Id())
	})
	m.topicPool.Put(topic)
	return nil
}

func init() {
	container.Container.Singleton(func() status.ISystemStatusController {
		return status.NewSystemStatusController()
	})
}
