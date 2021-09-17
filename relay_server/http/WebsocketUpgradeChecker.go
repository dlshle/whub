package http

import (
	"errors"
	"net/http"
	"wsdk/relay_server/container"
	"wsdk/relay_server/core/auth"
)

type IWebsocketUpgradeChecker interface {
	ShouldUpgradeProtocol(r *http.Request) error
}

type WebsocketUpgradeChecker struct {
	authController auth.IAuthController `$inject:""`
}

func NewWebsocketUpgradeChecker() IWebsocketUpgradeChecker {
	checker := &WebsocketUpgradeChecker{}
	err := container.Container.Fill(checker)
	if err != nil {
		panic(err)
	}
	return checker
}

func (c *WebsocketUpgradeChecker) ShouldUpgradeProtocol(r *http.Request) error {
	// deprecate the header token as not all ws client supports token in header
	// token := auth.GetTrimmedHTTPToken(r.Header)
	token := GetTokenFromQueryParameters(r)
	if token == "" {
		return errors.New("invalid auth token")
	}
	// validate token
	_, err := c.authController.ValidateToken(token)
	return err
}

func GetTokenFromQueryParameters(r *http.Request) string {
	return r.URL.Query().Get("token")
}
