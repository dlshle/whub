package topic

import (
	"sync"
	"wsdk/relay_server/container"
	"wsdk/relay_server/controllers"
)

const DefaultTopicCacheSize = 512

type CacheTopicStore struct {
	cacheSize int
	topics    map[string]*Topic
	pool      *sync.Pool
	lock      *sync.RWMutex
}

func NewCacheTopicStore(cacheSize int) ITopicStore {
	return &CacheTopicStore{
		cacheSize: cacheSize,
		topics:    make(map[string]*Topic),
		pool: &sync.Pool{
			New: func() interface{} {
				return &Topic{}
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

func (s *CacheTopicStore) set(id string, topic *Topic) {
	t := s.get(id)
	if t == nil {
		return
	}
	s.withWrite(func() {
		s.topics[id] = topic
	})
}

func (s *CacheTopicStore) Has(id string) (bool, controllers.IControllerError) {
	return s.get(id) != nil, nil
}

func (s *CacheTopicStore) Create(id string, creatorClientId string) (ret *Topic, err controllers.IControllerError) {
	s.withWrite(func() {
		if len(s.topics) >= s.cacheSize {
			err = NewTopicCacheSizeExceededError(s.cacheSize)
			return
		}
		t := s.pool.Get().(*Topic)
		t.Init(id, creatorClientId)
		ret = t
	})
	return
}

func (s *CacheTopicStore) Update(t *Topic) controllers.IControllerError {
	ret := s.get(t.Id())
	if ret == nil {
		_, err := s.Create(t.Id(), t.Creator())
		return err
	}
	s.set(t.Id(), t)
	return nil
}

func (s *CacheTopicStore) Get(id string) (*Topic, controllers.IControllerError) {
	if t := s.get(id); t != nil {
		return t, nil
	}
	return nil, NewTopicNotFoundError(id)
}

func (s *CacheTopicStore) Delete(id string) controllers.IControllerError {
	t := s.get(id)
	if t == nil {
		return NewTopicNotFoundError(id)
	}
	s.withWrite(func() {
		delete(s.topics, id)
	})
	s.pool.Put(t)
	return nil
}

func (s *CacheTopicStore) Topics() ([]*Topic, controllers.IControllerError) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	topics := make([]*Topic, 0, len(s.topics))
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
