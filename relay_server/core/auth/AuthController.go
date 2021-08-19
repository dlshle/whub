package auth

import (
	"errors"
	"fmt"
	"time"
	base_conn "wsdk/common/connection"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/messages"
	"wsdk/relay_server/client"
	"wsdk/relay_server/core/client_manager"
	"wsdk/relay_server/core/connection_manager"
)

const (
	AsyncConnTtl = time.Hour
	SyncConnTtl  = time.Minute * 30
	MaxTokenTtl  = time.Hour * 24
)

type IAuthController interface {
	ValidateRequestSource(conn connection.IConnection, request messages.IMessage) (bool, error)
}

type AuthController struct {
	clientManager client_manager.IClientManager         `$inject:""`
	connManager   connection_manager.IConnectionManager `$inject:""`
	store         ITokenStore
}

// ValidateRequestSource if returns true, nil => logged in; false, nil => not logged in; o/w credential check failure
func (c *AuthController) ValidateRequestSource(conn connection.IConnection, request messages.IMessage) (bool, error) {
	if base_conn.IsAsyncType(conn.ConnectionType()) {
		return c.validateAsyncConnRequest(conn, request)
	}
	return c.validateSyncConnRequest(conn, request)
}

func (c *AuthController) validateAsyncConnRequest(conn connection.IConnection, request messages.IMessage) (bool, error) {
	conns, err := c.connManager.GetConnectionsByClientId(request.From())
	if err != nil {
		return false, err
	}
	for i := range conns {
		if conns[i].Address() == conn.Address() {
			return true, nil
		}
	}
	return false, nil
}

func (c *AuthController) validateSyncConnRequest(conn connection.IConnection, request messages.IMessage) (bool, error) {
	authToken := request.From()
	if authToken == "" {
		return false, nil
	}
	client, err := c.clientManager.GetClient(request.From())
	if err != nil {
		return false, err
	}
	return c.validateToken(client.Id(), client.CKey(), authToken)
}

func (c *AuthController) validateToken(clientId string, clientCKey string, token string) (bool, error) {
	if c.checkTokenFromStore(token, clientId) == nil {
		return true, nil
	}
	err := c.checkToken(token, clientId, clientCKey)
	return err == nil, err
}

func (c *AuthController) checkTokenFromStore(token string, clientId string) error {
	clientIdFromToken, err := c.store.Get(token)
	if err == nil && clientIdFromToken == clientId {
		return nil
	}
	return err
}

func (c *AuthController) checkToken(token string, clientId string, clientCKey string) error {
	_, err := VerifyToken(token, func(claimMap map[string]interface{}) (string, error) {
		if claimMap["id"] == nil || claimMap["ttl"] == nil || claimMap["signTime"] == nil {
			return "", errors.New("missing token claims")
		}
		if claimMap["id"].(string) != clientId {
			return "", errors.New("mismatched client-id")
		}
		if claimMap["ttl"].(int) != 0 && c.isTokenExpired(claimMap["signTime"].(int64), claimMap["ttl"].(int)) {
			return "", errors.New("token expired")
		}
		return clientCKey, nil
	})
	return err
}

func (c *AuthController) isTokenExpired(signTimeNano int64, ttlInNano int) bool {
	expireTime := time.Unix(0, signTimeNano).Add(time.Nanosecond * time.Duration(ttlInNano))
	return time.Now().After(expireTime)
}

// Login only for un-authed clients
func (c *AuthController) Login(connType uint8, request messages.IMessage) (string, error) {
	if base_conn.IsAsyncType(connType) {
		return c.asyncConnLogin(request)
	}
	return c.syncConnLogin(request)
}

func (c *AuthController) getClientAndCheckCredential(request messages.IMessage) (*client.Client, error) {
	roleDesc, extraInfo, err := client_manager.UnmarshallClientDescriptor(request)
	if err != nil {
		return nil, err
	}
	client, err := c.clientManager.GetClient(roleDesc.Id)
	if err != nil {
		return nil, err
	}
	if client.CKey() != extraInfo.CKey {
		return nil, errors.New("invalid id or ckey")
	}
	return client, nil
}

func (c *AuthController) asyncConnLogin(request messages.IMessage) (string, error) {
	client, err := c.getClientAndCheckCredential(request)
	if err != nil {
		return "", err
	}
	return SignToken(client.Id(), client.CKey(), AsyncConnTtl, TokenTypeDefault)
}

func (c *AuthController) syncConnLogin(request messages.IMessage) (string, error) {
	client, err := c.getClientAndCheckCredential(request)
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
