package pubsub

import (
	"errors"
	"fmt"
	"sync"
)

const MaxSubscribersPerTopic = 128

type Topic struct {
	id             string
	creator        string
	subscribersMap map[string]bool
	lock           *sync.RWMutex
}

func NewTopic(id string, creatorId string) *Topic {
	topic := &Topic{
		id:             id,
		creator:        creatorId,
		subscribersMap: make(map[string]bool),
	}
	topic.subscribersMap[creatorId] = true
	return topic
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
		t.subscribersMap[subscriber] = false
	})
}

func (t *Topic) CheckAndAddSubscriber(subscriber string) error {
	if t.NumSubscribers() >= MaxSubscribersPerTopic {
		return errors.New(fmt.Sprintf("number of subscribers exceeded max subscribers count %d", MaxSubscribersPerTopic))
	}
	if t.HasSubscriber(subscriber) {
		return errors.New(fmt.Sprintf("subscriber %s has already subscriberd to topic %s", subscriber, t.id))
	}
	t.addSubscriber(subscriber)
	return nil
}

func (t *Topic) CheckAndRemoveSubscriber(subscriber string) error {
	if !t.HasSubscriber(subscriber) {
		return errors.New(fmt.Sprintf("subscriber %s is not subscriberd to topic %s", subscriber, t.id))
	}
	t.removeSubscriber(subscriber)
	return nil
}
