package auth

import (
	"errors"
	"fmt"
	"time"
	base_conn "wsdk/common/connection"
	"wsdk/common/logger"
	"wsdk/common/redis"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/roles"
	"wsdk/relay_server/client"
	"wsdk/relay_server/config"
	"wsdk/relay_server/container"
	"wsdk/relay_server/context"
	"wsdk/relay_server/core/client_manager"
	"wsdk/relay_server/core/connection_manager"
)

const (
	AsyncConnTtl = time.Hour
	SyncConnTtl  = time.Minute * 30
	MaxTokenTtl  = time.Hour * 24
)

type IAuthController interface {
	ValidateRequestSource(conn connection.IConnection, request messages.IMessage) (string, error)
	ValidateToken(token string) (string, error)
	Login(connType uint8, id, password string) (string, error)
	RefreshToken(token, clientId string, refreshTokenMessage RefreshTokenMessageBody) (string, error)
	RevokeToken(token string) error
}

type AuthController struct {
	clientManager client_manager.IClientManager         `$inject:""`
	connManager   connection_manager.IConnectionManager `$inject:""`
	store         ITokenStore
	logger        *logger.SimpleLogger
}

func NewAuthController() IAuthController {
	logger := context.Ctx.Logger().WithPrefix("[AuthController]")
	store := createTokenStore(logger)
	controller := &AuthController{
		logger: logger,
		store:  store,
	}
	if err := container.Container.Fill(controller); err != nil {
		panic(err)
	}
	return controller
}

func createTokenStore(logger *logger.SimpleLogger) ITokenStore {
	redisConfig := config.Config.DomainConfigs["authController"].Redis
	if redisConfig.Server == "" {
		logger.Println("init in memory store")
		return NewMemoryTokenStore()
	}
	store, err := NewRedisTokenStore(redisConfig.Server, redisConfig.Password)
	logger.Printf("init redis store with redis server %s", redisConfig.Server)
	if err != nil {
		logger.Printf("unable to create redis token store due to %s, will use in memory store", err.Error())
		store = NewMemoryTokenStore()
	}
	return store
}

// ValidateRequestSource if returns true, nil => logged in; false, nil => not logged in; o/w credential check failure
func (c *AuthController) ValidateRequestSource(conn connection.IConnection, request messages.IMessage) (string, error) {
	if base_conn.IsAsyncType(conn.ConnectionType()) {
		return c.validateAsyncConnRequest(conn, request)
	}
	return c.validateSyncConnRequest(conn, request)
}

func (c *AuthController) validateAsyncConnRequest(conn connection.IConnection, request messages.IMessage) (string, error) {
	conns, err := c.connManager.GetConnectionsByClientId(request.From())
	if err != nil {
		return "", err
	}
	for i := range conns {
		if conns[i].Address() == conn.Address() {
			return request.From(), nil
		}
	}
	// impossible case or connection dropped when/before conducting the request
	return "", nil
}

func (c *AuthController) validateSyncConnRequest(conn connection.IConnection, request messages.IMessage) (string, error) {
	authToken := request.From()
	if authToken == "" {
		return "", nil
	}
	return c.ValidateToken(authToken)
}

func (c *AuthController) ValidateToken(token string) (string, error) {
	if token == "" {
		return "", nil
	}
	clientId, err := c.checkTokenFromStore(token)
	if err != nil {
		return clientId, nil
	}
	return c.parseToken(token)
}

func (c *AuthController) checkTokenFromStore(token string) (string, error) {
	clientIdFromToken, err := c.store.Get(token)
	if err == nil || err.Error() == redis.ErrNotFoundStr {
		return "", nil
	}
	return clientIdFromToken, err
}

func (c *AuthController) parseToken(token string) (clientId string, err error) {
	_, err = VerifyToken(token, func(claim *TokenClaim) (string, error) {
		clientId = claim.ClientId
		if c.isTokenExpired(claim.ExpiresAt) {
			return "", errors.New("token expired")
		}
		client, err := c.clientManager.GetClient(clientId)
		if err != nil {
			return "", err
		}
		return client.CKey(), nil
	})
	return
}

func (c *AuthController) isTokenExpired(expireTime int64) bool {
	return time.Now().After(time.Unix(expireTime, 0))
}

// Login only for un-authed clients
func (c *AuthController) Login(connType uint8, id, password string) (string, error) {
	if base_conn.IsAsyncType(connType) {
		return c.asyncConnLogin(id, password)
	}
	return c.syncConnLogin(id, password)
}

func (c *AuthController) getClientAndCheckCredential(id, password string) (*client.Client, error) {
	client, err := c.clientManager.GetClient(id)
	if err != nil {
		return nil, err
	}
	if client.CKey() != password {
		return nil, errors.New("invalid id or password")
	}
	return client, nil
}

func (c *AuthController) asyncConnLogin(id, password string) (string, error) {
	client, err := c.getClientAndCheckCredential(id, password)
	if err != nil {
		return "", err
	}
	return c.loginAndCacheToken(client, AsyncConnTtl, TokenTypeDefault)
}

func (c *AuthController) syncConnLogin(id, password string) (string, error) {
	client, err := c.getClientAndCheckCredential(id, password)
	if err != nil {
		return "", err
	}
	return c.loginAndCacheToken(client, SyncConnTtl, TokenTypeDefault)
}

func (c *AuthController) loginAndCacheToken(client *client.Client, ttl time.Duration, tokenType uint8) (string, error) {
	clientId, clientCKey := client.Id(), client.CKey()
	// service role token can be valid for as much as 180 days
	if client.CType() == roles.ClientTypeService {
		ttl = time.Hour * 24 * 180
	}
	token, err := SignToken(clientId, clientCKey, ttl, tokenType)
	if err != nil {
		return "", err
	}
	if err = c.store.Put(token, clientId, ttl); err != nil {
		c.logger.Printf("cache token %s from %s to redis failed due to %s", token, clientId, err.Error())
	}
	return token, nil
}

// RefreshToken only available for authed client
func (c *AuthController) RefreshToken(token, clientId string, refreshTokenMessage RefreshTokenMessageBody) (string, error) {
	if refreshTokenMessage.Ttl < 0 || refreshTokenMessage.Ttl > MaxTokenTtl.Milliseconds() {
		return "", fmt.Errorf("invalid ttl %d. 0 < ttl < %d", refreshTokenMessage.Ttl, MaxTokenTtl.Milliseconds())
	}
	return c.refreshToken(token, clientId, time.Millisecond*time.Duration(refreshTokenMessage.Ttl))
}

func (c *AuthController) refreshToken(oldToken string, clientId string, ttl time.Duration) (string, error) {
	err := c.store.Revoke(oldToken)
	if err != nil {
		return "", err
	}
	client, err := c.clientManager.GetClient(clientId)
	if err != nil {
		return "", err
	}
	return SignToken(client.Id(), client.CKey(), ttl, TokenTypeDefault)
}

func (c *AuthController) RevokeToken(token string) error {
	return c.store.Revoke(token)
}

func Load() error {
	return container.Container.Singleton(func() IAuthController {
		return NewAuthController()
	})
}
