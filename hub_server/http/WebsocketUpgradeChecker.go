package http

import (
	"errors"
	"net/http"
	"whub/hub_server/module_base"
	"whub/hub_server/modules/auth"
)

type IWebsocketUpgradeChecker interface {
	ShouldUpgradeProtocol(r *http.Request) error
}

type WebsocketUpgradeChecker struct {
	authController auth.IAuthModule `module:""`
}

func NewWebsocketUpgradeChecker() IWebsocketUpgradeChecker {
	checker := &WebsocketUpgradeChecker{}
	err := module_base.Manager.AutoFill(checker)
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
