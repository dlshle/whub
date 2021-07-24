package topic

import (
	"sync"
	"wsdk/relay_server/controllers"
)

const MaxSubscribersPerTopic = 128

type Topic struct {
	id             string
	creator        string
	subscribersMap map[string]bool
	lock           *sync.RWMutex
}

type TopicDescriptor struct {
	Id          string   `json:"id"`
	Creator     string   `json:"creator"`
	Subscribers []string `json:"subscribers"`
}

func (t *Topic) Init(id string, creatorId string) {
	t.id = id
	t.creator = creatorId
	t.subscribersMap = make(map[string]bool)
	t.subscribersMap[creatorId] = true
}

func (t *Topic) withWrite(cb func()) {
	t.lock.Lock()
	defer t.lock.Unlock()
	cb()
}

func (t *Topic) Id() string {
	return t.id
}

func (t *Topic) Creator() string {
	return t.creator
}

func (t *Topic) NumSubscribers() int {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return len(t.subscribersMap)
}

func (t *Topic) HasSubscriber(id string) bool {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.subscribersMap[id]
}

func (t *Topic) Subscribers() []string {
	t.lock.RLock()
	defer t.lock.RUnlock()
	subscribers := make([]string, 0, t.NumSubscribers())
	for k := range t.subscribersMap {
		subscribers = append(subscribers, k)
	}
	return subscribers
}

func (t *Topic) addSubscriber(subscriber string) {
	t.withWrite(func() {
		t.subscribersMap[subscriber] = true
	})
}

func (t *Topic) removeSubscriber(subscriber string) {
	t.withWrite(func() {
		delete(t.subscribersMap, subscriber)
	})
}

func (t *Topic) CheckAndAddSubscriber(subscriber string) controllers.IControllerError {
	if t.NumSubscribers() >= MaxSubscribersPerTopic {
		return NewTopicMaxSubscribersExceededError(t.id, MaxSubscribersPerTopic)
	}
	if t.HasSubscriber(subscriber) {
		return NewTopicClientAlreadySubscribedError(t.id, subscriber)
	}
	t.addSubscriber(subscriber)
	return nil
}

func (t *Topic) CheckAndRemoveSubscriber(subscriber string) controllers.IControllerError {
	if !t.subscribersMap[subscriber] {
		return NewTopicClientNotValidSubscriberError(t.id, subscriber)
	}
	t.removeSubscriber(subscriber)
	return nil
}

func (t *Topic) Describe() TopicDescriptor {
	return TopicDescriptor{
		Id:          t.Id(),
		Creator:     t.Creator(),
		Subscribers: t.Subscribers(),
	}
}
