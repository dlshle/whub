package topic

import (
	"errors"
	"fmt"
	"sync"
)

type CacheTopicStore struct {
	cacheSize int
	topics    map[string]*Topic
	pool      *sync.Pool
	lock      *sync.RWMutex
}

func (s *CacheTopicStore) withWrite(cb func()) {
	s.lock.Lock()
	defer s.lock.Unlock()
	cb()
}

func (s *CacheTopicStore) get(id string) *Topic {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.topics[id]
}

func (s *CacheTopicStore) size() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return len(s.topics)
}

func (s *CacheTopicStore) update(id string, topic Topic) {
	t := s.get(id)
	if t == nil {
		return
	}
	s.withWrite(func() {
		// TODO how????
		// need to separate topic and actions
	})
}

func (s *CacheTopicStore) Has(id string) (bool, error) {
	return s.get(id) != nil, nil
}

func (s *CacheTopicStore) Create(id string, creatorClientId string) (topic Topic, err error) {
	s.withWrite(func() {
		if len(s.topics) >= s.cacheSize {
			err = errors.New(fmt.Sprintf("topic size exceeding cache size(%d)", s.cacheSize))
			return
		}
		t := s.pool.Get().(*Topic)
		t.Init(id, creatorClientId)
		topic = *t
	})
	return
}

func (s *CacheTopicStore) Update(topic Topic) error {
	t := s.get(topic.id)
	if t == nil {
		_, err := s.Create(topic.id, topic.creator)
		return err
	}
	s.set(topic.id, topic)
}

func (s *CacheTopicStore) Get(id string) (Topic, error) {

}

func (s *CacheTopicStore) Delete(id string) (Topic, error) {

}
