package auth

import (
	"fmt"
	"time"
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
}

type RedisTokenStore struct {
	redis *redis.RedisClient
}

func NewRedisTokenStore(serverAddr, passwd string) ITokenStore {
	redis := redis.NewRedisClient(serverAddr, passwd, 5)
	if err := redis.Ping(); err != nil {
		panic("redis token store init failed")
	}
	return &RedisTokenStore{
		redis: redis,
	}
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
