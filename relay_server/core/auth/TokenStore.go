package auth

import (
	"fmt"
	"time"
	"wsdk/common/redis"
)

const TokenStorePrefix = "token-"

type ITokenStore interface {
	Put(token string, clientId string, ttl int) error
	Get(token string) (string, error)
}

type TokenStore struct {
	redis *redis.RedisClient
}

func (s *TokenStore) assembleKey(key string) string {
	return fmt.Sprintf("%s%s", TokenStorePrefix, key)
}

func (s *TokenStore) Put(token string, clientId string, ttl int) error {
	if ttl == 0 {
		return s.redis.Set(s.assembleKey(token), clientId)
	}
	return s.redis.SetWithExp(s.assembleKey(token), clientId, time.Duration(ttl))
}

func (s *TokenStore) Get(token string) (string, error) {
	return s.redis.Get(s.assembleKey(token))
}
