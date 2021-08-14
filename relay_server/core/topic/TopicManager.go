package topic

import (
	"wsdk/common/logger"
	"wsdk/relay_common/messages"
	"wsdk/relay_server/container"
	"wsdk/relay_server/context"
	"wsdk/relay_server/core"
	"wsdk/relay_server/events"
)

type TopicManager struct {
	store  ITopicStore `$inject:""`
	logger *logger.SimpleLogger
}

type ITopicManager interface {
	GetTopic(id string) (topic Topic, err core.IControllerError)
	HasTopic(id string) (h bool, err core.IControllerError)
	SubscribeClientToTopic(clientId string, topicId string) (err core.IControllerError)
	UnSubscribeClientToTopic(clientId string, topicId string) (err core.IControllerError)
	GetSubscriberIds(topicId string) (ids []string, err core.IControllerError)
	GetAllDescribedTopics() (desc []TopicDescriptor, err core.IControllerError)
	CreateTopic(topicId string, creatorClientId string) (topic Topic, err core.IControllerError)
	RemoveTopic(topicId string, requestClientId string) (err core.IControllerError)
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

func (m *TopicManager) get(id string) (*Topic, core.IControllerError) {
	return m.store.Get(id)
}

func (m *TopicManager) has(id string) (bool, core.IControllerError) {
	return m.store.Has(id)
}

func (m *TopicManager) topics() ([]*Topic, core.IControllerError) {
	return m.store.Topics()
}

func (m *TopicManager) delete(id string) core.IControllerError {
	return m.store.Delete(id)
}

func (m *TopicManager) create(topicId string, creatorId string) (*Topic, core.IControllerError) {
	return m.store.Create(topicId, creatorId)
}

func (m *TopicManager) GetTopic(id string) (topic Topic, err core.IControllerError) {
	defer logger.LogError(m.logger, "GetTopic", err)
	t, err := m.get(id)
	return *t, err
}

func (m *TopicManager) HasTopic(id string) (h bool, err core.IControllerError) {
	defer logger.LogError(m.logger, "HasTopic", err)
	h, err = m.has(id)
	return
}

func (m *TopicManager) SubscribeClientToTopic(clientId string, topicId string) (err core.IControllerError) {
	defer logger.LogError(m.logger, "SubscribeClientToTopic", err)
	defer func() {
		if err == nil {
			m.logger.Printf("client %s has subscribed to topic %s", clientId, topicId)
		}
	}()
	topic, err := m.get(topicId)
	if err != nil {
		return err
	}
	return topic.CheckAndAddSubscriber(clientId)
}

func (m *TopicManager) UnSubscribeClientToTopic(clientId string, topicId string) (err core.IControllerError) {
	defer logger.LogError(m.logger, "UnSubscribeClientToTopic", err)
	defer func() {
		if err == nil {
			m.logger.Printf("client %s has unsubscribed from topic %s", clientId, topicId)
		}
	}()
	topic, err := m.get(topicId)
	if err != nil {
		return err
	}
	return topic.CheckAndRemoveSubscriber(clientId)
}

func (m *TopicManager) GetSubscriberIds(topicId string) (ids []string, err core.IControllerError) {
	defer logger.LogError(m.logger, "GetSubscriberIds", err)
	topic, err := m.get(topicId)
	if err != nil {
		return nil, err
	}
	return topic.Subscribers(), nil
}

func (m *TopicManager) GetAllDescribedTopics() (desc []TopicDescriptor, err core.IControllerError) {
	defer logger.LogError(m.logger, "GetAllDescribedTopics", err)
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

func (m *TopicManager) CreateTopic(topicId string, creatorClientId string) (topic Topic, err core.IControllerError) {
	defer logger.LogError(m.logger, "CreateTopic", err)
	t, err := m.create(topicId, creatorClientId)
	if err != nil {
		return Topic{}, err
	}
	m.logger.Printf("new topic %s has been created by %s", topicId, creatorClientId)
	return *t, err
}

// TODO caller needs to get the subscribers first to notify them and then call this function
func (m *TopicManager) RemoveTopic(topicId string, requestClientId string) (err core.IControllerError) {
	defer logger.LogError(m.logger, "RemoveTopic", err)
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
		return NewTopicClientInsufficientPermissionError(topic.Id(), requestClientId, "creator")
	}
	return m.delete(topicId)
}

func init() {
	container.Container.Singleton(func() ITopicManager {
		return NewTopicManager()
	})
}
