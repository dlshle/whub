package store

import (
	"sync"
	"wsdk/relay_server/container"
	"wsdk/relay_server/controllers/topic"
	error2 "wsdk/relay_server/controllers/topic/error"
)

const DefaultTopicCacheSize = 512

type CacheTopicStore struct {
	cacheSize int
	topics    map[string]*topic.Topic
	pool      *sync.Pool
	lock      *sync.RWMutex
}

func NewCacheTopicStore(cacheSize int) ITopicStore {
	return &CacheTopicStore{
		cacheSize: cacheSize,
		topics:    make(map[string]*topic.Topic),
		pool: &sync.Pool{
			New: func() interface{} {
				return &topic.Topic{}
			},
		},
		lock: new(sync.RWMutex),
	}
}

func (s *CacheTopicStore) withWrite(cb func()) {
	s.lock.Lock()
	defer s.lock.Unlock()
	cb()
}

func (s *CacheTopicStore) get(id string) *topic.Topic {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.topics[id]
}

func (s *CacheTopicStore) size() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return len(s.topics)
}

func (s *CacheTopicStore) set(id string, topic *topic.Topic) {
	t := s.get(id)
	if t == nil {
		return
	}
	s.withWrite(func() {
		s.topics[id] = topic
	})
}

func (s *CacheTopicStore) Has(id string) (bool, error2.ITopicError) {
	return s.get(id) != nil, nil
}

func (s *CacheTopicStore) Create(id string, creatorClientId string) (topic *topic.Topic, err error2.ITopicError) {
	s.withWrite(func() {
		if len(s.topics) >= s.cacheSize {
			err = topic.NewTopicCacheSizeExceededError(s.cacheSize)
			return
		}
		t := s.pool.Get().(*topic.Topic)
		t.Init(id, creatorClientId)
		topic = t
	})
	return
}

func (s *CacheTopicStore) Update(topic *topic.Topic) error2.ITopicError {
	t := s.get(topic.id)
	if t == nil {
		_, err := s.Create(topic.id, topic.creator)
		return err
	}
	s.set(topic.id, topic)
	return nil
}

func (s *CacheTopicStore) Get(id string) (*topic.Topic, error2.ITopicError) {
	if t := s.get(id); t != nil {
		return t, nil
	}
	return nil, error2.NewTopicNotFoundError(id)
}

func (s *CacheTopicStore) Delete(id string) error2.ITopicError {
	t := s.get(id)
	if t == nil {
		return error2.NewTopicNotFoundError(id)
	}
	s.withWrite(func() {
		delete(s.topics, id)
	})
	s.pool.Put(t)
	return nil
}

func (s *CacheTopicStore) Topics() ([]*topic.Topic, error2.ITopicError) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	topics := make([]*topic.Topic, 0, len(s.topics))
	for _, t := range s.topics {
		topics = append(topics, t)
	}
	return topics, nil
}

func init() {
	container.Container.Singleton(func() ITopicStore {
		return NewCacheTopicStore(DefaultTopicCacheSize)
	})
}
