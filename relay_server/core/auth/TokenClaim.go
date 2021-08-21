package auth

import (
	"github.com/golang-jwt/jwt"
	"time"
	"wsdk/relay_server/context"
)

type TokenPayload struct {
	ClientId string `json:"ClientId"`
}

type TokenClaim struct {
	jwt.StandardClaims
	TokenPayload
}

func NewTokenClaim(clientId string, ttl time.Duration) *TokenClaim {
	return &TokenClaim{
		jwt.StandardClaims{
			Issuer:    context.Ctx.Server().Id(),
			ExpiresAt: time.Now().Add(ttl).Unix(),
		},
		TokenPayload{
			ClientId: clientId,
		},
	}
}
