package topic

import (
	"wsdk/common/logger"
	"wsdk/common/utils"
	"wsdk/relay_common/messages"
	"wsdk/relay_server/container"
	"wsdk/relay_server/context"
	error2 "wsdk/relay_server/controllers/topic/error"
	store2 "wsdk/relay_server/controllers/topic/store"
	"wsdk/relay_server/events"
)

type TopicManager struct {
	store  store2.ITopicStore `$inject:""`
	logger *logger.SimpleLogger
}

type ITopicManager interface {
	GetTopic(id string) (topic Topic, err error2.ITopicError)
	HasTopic(id string) (h bool, err error2.ITopicError)
	SubscribeClientToTopic(clientId string, topicId string) (err error2.ITopicError)
	UnSubscribeClientToTopic(clientId string, topicId string) (err error2.ITopicError)
	GetSubscriberIds(topicId string) (ids []string, err error2.ITopicError)
	GetAllDescribedTopics() (desc []TopicDescriptor, err error2.ITopicError)
	CreateTopic(topicId string, creatorClientId string) (topic Topic, err error2.ITopicError)
	RemoveTopic(topicId string, requestClientId string) (err error2.ITopicError)
}

func NewTopicManager() ITopicManager {
	m := &TopicManager{
		logger: context.Ctx.Logger().WithPrefix("[TopicManager]"),
	}
	err := container.Container.Fill(m)
	if err != nil {
		panic(err)
	}
	return m
}

func (m *TopicManager) initNotificationHandlers() {
	events.OnEvent(events.EventClientDisconnected, func(e *messages.Message) {
		clientId := string(e.Payload()[:])
		topics, err := m.topics()
		if err != nil {
			m.logger.Printf("ITopicError while handling %s event due to %s", events.EventClientDisconnected, err.Error())
			return
		}
		for _, t := range topics {
			t.CheckAndRemoveSubscriber(clientId)
		}
	})
}

func (m *TopicManager) get(id string) (*Topic, error2.ITopicError) {
	return m.store.Get(id)
}

func (m *TopicManager) has(id string) (bool, error2.ITopicError) {
	return m.store.Has(id)
}

func (m *TopicManager) topics() ([]*Topic, error2.ITopicError) {
	return m.store.Topics()
}

func (m *TopicManager) delete(id string) error2.ITopicError {
	return m.store.Delete(id)
}

func (m *TopicManager) create(topicId string, creatorId string) (*Topic, error2.ITopicError) {
	return m.store.Create(topicId, creatorId)
}

func (m *TopicManager) GetTopic(id string) (topic Topic, err error2.ITopicError) {
	defer utils.LogError(m.logger, "GetTopic", err)
	t, err := m.get(id)
	return *t, err
}

func (m *TopicManager) HasTopic(id string) (h bool, err error2.ITopicError) {
	defer utils.LogError(m.logger, "HasTopic", err)
	h, err = m.has(id)
	return
}

func (m *TopicManager) SubscribeClientToTopic(clientId string, topicId string) (err error2.ITopicError) {
	defer utils.LogError(m.logger, "SubscribeClientToTopic", err)
	defer func() {
		if err == nil {
			m.logger.Printf("client_manager %s has subscribed to topic %s", clientId, topicId)
		}
	}()
	topic, err := m.get(topicId)
	if err != nil {
		return err
	}
	return topic.CheckAndAddSubscriber(clientId)
}

func (m *TopicManager) UnSubscribeClientToTopic(clientId string, topicId string) (err error2.ITopicError) {
	defer utils.LogError(m.logger, "UnSubscribeClientToTopic", err)
	defer func() {
		if err == nil {
			m.logger.Printf("client_manager %s has unsubscribed from topic %s", clientId, topicId)
		}
	}()
	topic, err := m.get(topicId)
	if err != nil {
		return err
	}
	return topic.CheckAndRemoveSubscriber(clientId)
}

func (m *TopicManager) GetSubscriberIds(topicId string) (ids []string, err error2.ITopicError) {
	defer utils.LogError(m.logger, "GetSubscriberIds", err)
	topic, err := m.get(topicId)
	if err != nil {
		return nil, err
	}
	return topic.Subscribers(), nil
}

func (m *TopicManager) GetAllDescribedTopics() (desc []TopicDescriptor, err error2.ITopicError) {
	defer utils.LogError(m.logger, "GetAllDescribedTopics", err)
	topics, err := m.topics()
	if err != nil {
		return nil, err
	}
	topicDescriptors := make([]TopicDescriptor, 0, len(topics))
	for i := range topics {
		topicDescriptors[i] = topics[i].Describe()
	}
	return topicDescriptors, nil
}

func (m *TopicManager) CreateTopic(topicId string, creatorClientId string) (topic Topic, err error2.ITopicError) {
	defer utils.LogError(m.logger, "CreateTopic", err)
	t, err := m.create(topicId, creatorClientId)
	if err != nil {
		return Topic{}, err
	}
	m.logger.Printf("new topic %s has been created by %s", topicId, creatorClientId)
	return *t, err
}

// TODO caller needs to get the subscribers first to notify them and then call this function
func (m *TopicManager) RemoveTopic(topicId string, requestClientId string) (err error2.ITopicError) {
	defer utils.LogError(m.logger, "RemoveTopic", err)
	defer func() {
		if err == nil {
			m.logger.Printf("topic %s has been removed", topicId)
		}
	}()
	topic, err := m.get(topicId)
	if err != nil {
		return err
	}
	if topic.Creator() != requestClientId {
		return error2.NewTopicClientInsufficientPermissionError(topic.Id(), requestClientId, "creator")
	}
	return m.delete(topicId)
}

func init() {
	container.Container.Singleton(func() ITopicManager {
		return NewTopicManager()
	})
}
