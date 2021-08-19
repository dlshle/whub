package auth

import (
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt"
	"time"
	"wsdk/relay_server/context"
)

var tokenSignMethodMap map[uint8]func(clientId string, clientCKey string, ttl int64) (string, error)

func init() {
	tokenSignMethodMap = make(map[uint8]func(clientId string, clientCKey string, ttl int64) (string, error))
	tokenSignMethodMap[TokenTypeDefault] = signDefaultToken
	tokenSignMethodMap[TokenTypePermanent] = signPermanentToken
}

const (
	TokenTypeDefault   = 0
	TokenTypePermanent = 1
)

func SignToken(clientId string, clientCKey string, ttlInNano time.Duration, tokenType uint8) (string, error) {
	return tokenSignMethodMap[tokenType](clientId, clientCKey, ttlInNano.Nanoseconds())
}

func signDefaultToken(clientId string, clientCKey string, ttl int64) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":       clientId,
		"ttl":      ttl,
		"signTime": time.Now(),
		"issuer":   context.Ctx.Server().Id(),
	})
	return token.SignedString(([]byte)(clientCKey))
}

func signPermanentToken(clientId string, clientCKey string, ttl int64) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":       clientId,
		"signTime": time.Now().UnixNano(),
		"ttl":      0,
		"issuer":   context.Ctx.Server().Id(),
	})
	return token.SignedString(([]byte)(clientCKey))
}

func ParseToken(token string) (*jwt.Token, error) {
	return jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return context.Ctx.SignKey(), nil
	})
}

func VerifyToken(stringToken string, verifyCallback func(map[string]interface{}) (string, error)) (*jwt.Token, error) {
	return jwt.Parse(stringToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		claimMap, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return nil, errors.New("unable to convert claim to map")
		}
		cKey, err := verifyCallback(claimMap)
		if err != nil {
			return nil, err
		}
		return ([]byte)(cKey), nil
	})
}
