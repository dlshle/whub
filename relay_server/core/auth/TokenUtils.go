package auth

import (
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt"
	"time"
)

var tokenSignMethodMap map[uint8]func(clientId string, clientCKey string, ttl time.Duration) (string, error)

func init() {
	tokenSignMethodMap = make(map[uint8]func(clientId string, clientCKey string, ttl time.Duration) (string, error))
	tokenSignMethodMap[TokenTypeDefault] = signDefaultToken
	tokenSignMethodMap[TokenTypePermanent] = signPermanentToken
}

const (
	TokenTypeDefault   = 0
	TokenTypePermanent = 1
)

func SignToken(clientId string, clientCKey string, ttlInNano time.Duration, tokenType uint8) (string, error) {
	return tokenSignMethodMap[tokenType](clientId, clientCKey, ttlInNano)
}

func signDefaultToken(clientId string, clientCKey string, ttl time.Duration) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, NewTokenClaim(clientId, ttl))
	return token.SignedString(([]byte)(clientCKey))
}

func signPermanentToken(clientId string, clientCKey string, ttl time.Duration) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, NewTokenClaim(clientId, ttl))
	return token.SignedString(([]byte)(clientCKey))
}

func VerifyToken(stringToken string, verifyCallback func(claim *TokenClaim) (string, error)) (*jwt.Token, error) {
	return jwt.ParseWithClaims(stringToken, &TokenClaim{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		tokenClaim, ok := token.Claims.(*TokenClaim)
		if !ok {
			return nil, errors.New("unable to convert claim to map")
		}
		cKey, err := verifyCallback(tokenClaim)
		if err != nil {
			return nil, err
		}
		return ([]byte)(cKey), nil
	})
}
