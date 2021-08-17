package redis

const (
	CachePolicyWriteThrough = 1
	CachePolicyWriteBack    = 2
)

type ISingleEntityStore interface {
	Get(id string) (interface{}, error)
	Update(id string, value interface{}) error
	Create(id string, value interface{}) error
	Delete(id string) error
	ToHashMap(interface{}) (map[string]interface{}, error)
	ToEntityType(map[string]string) (interface{}, error)
}

type RedisCachedStore struct {
	store                 ISingleEntityStore
	cache                 *RedisClient
	cacheOnCreate         bool
	skipErrOnCacheFailure bool
	writePolicy           uint8
}

func NewRedisCachedStore(store ISingleEntityStore, cache *RedisClient, cacheOnCreate bool, skipErrOnCacheFailure bool, writePolicy uint8) *RedisCachedStore {
	if writePolicy > CachePolicyWriteBack {
		writePolicy = CachePolicyWriteBack
	}
	return &RedisCachedStore{
		store:                 store,
		cache:                 cache,
		cacheOnCreate:         cacheOnCreate,
		skipErrOnCacheFailure: skipErrOnCacheFailure,
		writePolicy:           writePolicy,
	}
}

func (s *RedisCachedStore) Get(id string) (entity interface{}, err error) {
	var m map[string]string
	m, err = s.cache.HGet(id)
	if err != nil && err.Error() != "not found" && err != nil {
		return
	}
	if err != nil && err.Error() == "not found" {
		entity, err = s.store.Get(id)
		if s.skipErrOnCacheFailure {
			err = nil
		}
		return
	}
	return s.store.ToEntityType(m)
}

func (s *RedisCachedStore) Update(id string, value interface{}) error {
	m, err := s.store.ToHashMap(value)
	if err != nil {
		return err
	}
	switch s.writePolicy {
	case CachePolicyWriteThrough:
		return s.writeThroughSet(id, value, m)
	default:
		return s.writeBackSet(id, value, m)
	}
}

func (s *RedisCachedStore) writeThroughSet(id string, entity interface{}, m map[string]interface{}) (err error) {
	return s.writeThroughAction(func() error { return s.store.Update(id, entity) }, func() error { return s.cache.HSet(id, m) })
}

func (s *RedisCachedStore) writeBackSet(id string, entity interface{}, m map[string]interface{}) (err error) {
	return s.writeBackAction(func() error { return s.store.Update(id, entity) }, func() error { return s.cache.HSet(id, m) })
}

func (s *RedisCachedStore) writeThroughAction(storeAction func() error, cacheAction func() error) error {
	if err := cacheAction(); err != nil {
		return err
	}
	return storeAction()
}

func (s *RedisCachedStore) writeBackAction(storeAction func() error, cacheAction func() error) error {
	if err := storeAction(); err != nil {
		return err
	}
	return cacheAction()
}

func (s *RedisCachedStore) Create(id string, value interface{}) (err error) {
	if err = s.store.Create(id, value); err != nil {
		return
	}
	m, err := s.store.ToHashMap(value)
	if err != nil {
		return err
	}
	if s.cacheOnCreate {
		return s.cache.HSet(id, m)
	}
	return nil
}

func (s *RedisCachedStore) cacheSafeDelete(key string) error {
	err := s.Delete(key)
	if err != nil && err.Error() == "not found" {
		return nil
	}
	return err
}

func (s *RedisCachedStore) Delete(id string) error {
	switch s.writePolicy {
	case CachePolicyWriteThrough:
		return s.writeThroughAction(func() error { return s.store.Delete(id) }, func() error { return s.cacheSafeDelete(id) })
	default:
		return s.writeBackAction(func() error { return s.store.Delete(id) }, func() error { return s.cacheSafeDelete(id) })
	}
}
