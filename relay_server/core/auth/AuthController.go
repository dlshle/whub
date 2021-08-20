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
	"wsdk/relay_server/client"
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
}

type AuthController struct {
	clientManager client_manager.IClientManager         `$inject:""`
	connManager   connection_manager.IConnectionManager `$inject:""`
	store         ITokenStore
	logger        *logger.SimpleLogger
}

func NewAuthController() IAuthController {
	store := NewRedisTokenStore(RedisAddr, RedisPass)
	controller := &AuthController{
		logger: context.Ctx.Logger().WithPrefix("[AuthController]"),
		store:  store,
	}
	if err := container.Container.Fill(controller); err != nil {
		panic(err)
	}
	return controller
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
	_, err = VerifyToken(token, func(claimMap map[string]interface{}) (string, error) {
		if claimMap["id"] == nil || claimMap["ttl"] == nil || claimMap["signTime"] == nil {
			return "", errors.New("missing token claims")
		}
		// TODO remove later
		c.logger.Println("claim map: ", claimMap)
		/*
			claim map:  map[id:test2 issuer:test signTime:1.629488664892019e+18 ttl:1.8e+12]
		*/
		if claimMap["ttl"].(int) != 0 && c.isTokenExpired(claimMap["signTime"].(int64), claimMap["ttl"].(int)) {
			return "", errors.New("token expired")
		}
		clientId = claimMap["id"].(string)
		client, err := c.clientManager.GetClient(clientId)
		if err != nil {
			return "", err
		}
		return client.CKey(), nil
	})
	return
}

func (c *AuthController) isTokenExpired(signTimeNano int64, ttlInNano int) bool {
	expireTime := time.Unix(0, signTimeNano).Add(time.Nanosecond * time.Duration(ttlInNano))
	return time.Now().After(expireTime)
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
	return SignToken(client.Id(), client.CKey(), AsyncConnTtl, TokenTypeDefault)
}

func (c *AuthController) syncConnLogin(id, password string) (string, error) {
	client, err := c.getClientAndCheckCredential(id, password)
	if err != nil {
		return "", err
	}
	return SignToken(client.Id(), client.CKey(), SyncConnTtl, TokenTypeDefault)
}

// RefreshToken only available for authed client
func (c *AuthController) RefreshToken(request messages.IMessage) (string, error) {
	refreshTokenMessage, err := UnmarshallRefreshTokenMessageBody(request.Payload())
	if err != nil {
		return "", err
	}
	if refreshTokenMessage.Ttl < 0 || refreshTokenMessage.Ttl > MaxTokenTtl.Nanoseconds() {
		return "", fmt.Errorf("invalid ttl %d. 0 < ttl < %d", refreshTokenMessage.Ttl, MaxTokenTtl.Nanoseconds())
	}
	return c.refreshToken(request.From(), time.Nanosecond*time.Duration(refreshTokenMessage.Ttl))
}

func (c *AuthController) refreshToken(clientId string, ttl time.Duration) (string, error) {
	client, err := c.clientManager.GetClient(clientId)
	if err != nil {
		return "", err
	}
	return SignToken(client.Id(), client.CKey(), ttl, TokenTypeDefault)
}

func init() {
	container.Container.Singleton(func() IAuthController {
		return NewAuthController()
	})
}
