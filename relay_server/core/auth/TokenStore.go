package auth

import (
	"errors"
	"fmt"
	"sync"
	"time"
	"wsdk/common/ctimer"
	"wsdk/common/redis"
)

const TokenStorePrefix = "token-"

const (
	RedisAddr = "192.168.0.132:6379"
	RedisPass = "19950416"
)

type ITokenStore interface {
	Put(token string, clientId string, ttl time.Duration) error
	Get(token string) (string, error)
	Revoke(token string) error
}

type RedisTokenStore struct {
	redis *redis.RedisClient
}

func NewRedisTokenStore(serverAddr, passwd string) (ITokenStore, error) {
	redis := redis.NewRedisClient(serverAddr, passwd, 5)
	if err := redis.Ping(); err != nil {
		return nil, err
	}
	return &RedisTokenStore{
		redis: redis,
	}, nil
}

func (s *RedisTokenStore) assembleKey(key string) string {
	return fmt.Sprintf("%s%s", TokenStorePrefix, key)
}

func (s *RedisTokenStore) Put(token string, clientId string, ttl time.Duration) error {
	if ttl == 0 {
		return s.redis.Set(s.assembleKey(token), clientId)
	}
	return s.redis.SetWithExp(s.assembleKey(token), clientId, ttl)
}

func (s *RedisTokenStore) Get(token string) (string, error) {
	return s.redis.Get(s.assembleKey(token))
}

func (s *RedisTokenStore) Revoke(token string) error {
	return s.redis.Delete(token)
}

type inMemoryTokenInfo struct {
	clientId   string
	expireDate time.Time
}

type MemoryTokenStore struct {
	clientTokens map[string]*inMemoryTokenInfo
	lock         *sync.RWMutex
	timer        ctimer.ICTimer
}

func NewMemoryTokenStore() ITokenStore {
	store := &MemoryTokenStore{
		clientTokens: make(map[string]*inMemoryTokenInfo),
		lock:         new(sync.RWMutex),
	}
	// every minute to check if there's expired token to remove
	timer := ctimer.New(time.Minute, store.timerJob)
	store.timer = timer
	timer.Repeat()
	return store
}

func (s *MemoryTokenStore) withWrite(cb func()) {
	s.lock.Lock()
	defer s.lock.Unlock()
	cb()
}

func (s *MemoryTokenStore) timerJob() {
	var toRevokeTokens []string
	checkTime := time.Now()
	s.lock.RLock()
	for k, v := range s.clientTokens {
		if v.expireDate.Before(checkTime) {
			toRevokeTokens = append(toRevokeTokens, k)
		}
	}
	s.lock.RUnlock()
	s.withWrite(func() {
		for _, token := range toRevokeTokens {
			delete(s.clientTokens, token)
		}
	})
}

func (s *MemoryTokenStore) Put(token string, clientId string, ttl time.Duration) error {
	s.withWrite(func() {
		s.clientTokens[token] = &inMemoryTokenInfo{
			clientId:   clientId,
			expireDate: time.Now().Add(ttl),
		}
	})
	return nil
}

func (s *MemoryTokenStore) Get(token string) (string, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	info := s.clientTokens[token]
	if info == nil {
		return "", errors.New("can not find token")
	}
	return info.clientId, nil
}

func (s *MemoryTokenStore) Revoke(token string) (err error) {
	s.withWrite(func() {
		if s.clientTokens[token] == nil {
			err = errors.New("can not find token")
			return
		}
		delete(s.clientTokens, token)
	})
	return
}
