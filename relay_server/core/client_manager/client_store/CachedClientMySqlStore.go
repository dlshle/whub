package client_store

import (
	"errors"
	"strconv"
	"wsdk/common/redis"
	"wsdk/relay_server/client"
)

var InCompatibleError error

func init() {
	InCompatibleError = errors.New("incompatible type conversion of Client")
}

type CachedClientMySqlStore struct {
	cachedStore *redis.RedisCachedStore
	redisClient *redis.RedisClient
	mySqlStore  *ClientMySqlStore
}

func NewCachedClientMySqlStore() *CachedClientMySqlStore {
	return &CachedClientMySqlStore{}
}

func (s *CachedClientMySqlStore) Init(fullDBUri, username, password, dbname string, redisAddr, redisPass string) error {
	s.mySqlStore = NewMySqlClientStore()
	if err := s.mySqlStore.Init(fullDBUri, username, password, dbname); err != nil {
		return err
	}
	s.redisClient = redis.NewRedisClient(redisAddr, redisPass, 5)
	if err := s.redisClient.Ping(); err != nil {
		return err
	}
	s.cachedStore = redis.NewRedisCachedStore(NewMySqlStoreCacheAdaptor(s.mySqlStore), s.redisClient, true, false, redis.CachePolicyWriteBack)
	return nil
}

func (s *CachedClientMySqlStore) Get(id string) (*client.Client, error) {
	iClient, err := s.cachedStore.Get(id)
	if err != nil {
		return nil, err
	}
	client, ok := iClient.(*client.Client)
	if !ok {
		return nil, InCompatibleError
	}
	return client, nil
}

func (s *CachedClientMySqlStore) GetAll() ([]*client.Client, error) {
	return s.mySqlStore.GetAll()
}

func (s *CachedClientMySqlStore) Create(client *client.Client) error {
	return s.cachedStore.Create(client.Id(), client)
}

func (s *CachedClientMySqlStore) Update(client *client.Client) error {
	return s.cachedStore.Update(client.Id(), client)
}

func (s *CachedClientMySqlStore) Has(id string) (bool, error) {
	data, err := s.Get(id)
	return data != nil, err
}

func (s *CachedClientMySqlStore) Delete(id string) error {
	return s.cachedStore.Delete(id)
}

func (s *CachedClientMySqlStore) Find(query *DClientQuery) ([]*client.Client, error) {
	return s.mySqlStore.Find(query)
}

type MySqlStoreCacheAdaptor struct {
	mySqlStore *ClientMySqlStore
}

func NewMySqlStoreCacheAdaptor(store *ClientMySqlStore) *MySqlStoreCacheAdaptor {
	return &MySqlStoreCacheAdaptor{store}
}

func (a *MySqlStoreCacheAdaptor) Get(id string) (interface{}, error) {
	return a.mySqlStore.Get(id)
}

func (a *MySqlStoreCacheAdaptor) Update(id string, value interface{}) error {
	client, ok := value.(*client.Client)
	if !ok {
		return errors.New("incompatible client type")
	}
	return a.mySqlStore.Update(client)
}

func (a *MySqlStoreCacheAdaptor) Create(id string, value interface{}) error {
	client, ok := value.(*client.Client)
	if !ok {
		return errors.New("incompatible client type")
	}
	return a.mySqlStore.Create(client)
}

func (a *MySqlStoreCacheAdaptor) Delete(id string) error {
	return a.mySqlStore.Delete(id)
}

func (a *MySqlStoreCacheAdaptor) ToHashMap(iClient interface{}) (map[string]interface{}, error) {
	client, ok := iClient.(*client.Client)
	if !ok {
		return nil, errors.New("incompatible client type")
	}
	m := make(map[string]interface{})
	m["id"] = client.Id()
	m["description"] = client.Description()
	m["type"] = client.Type()
	m["cKey"] = client.CKey()
	m["cType"] = client.CType()
	m["pScope"] = client.PScope()
	return m, nil
}

func (a *MySqlStoreCacheAdaptor) ToEntityType(m map[string]string) (entity interface{}, err error) {
	id := m["id"]
	desc := m["description"]
	cKey := m["cKey"]
	cType, err := strconv.Atoi(m["cType"])
	if err != nil {
		return nil, err
	}
	pScope, err := strconv.Atoi(m["pScope"])
	if err != nil {
		return nil, err
	}
	return client.NewClient(id, desc, cType, cKey, pScope), nil
}
