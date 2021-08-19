package http

import (
	"errors"
	"net/http"
	"wsdk/relay_server/core/auth"
)

// TODO how do we make this happen before the upgrade?
type WebsocketUpgradeChecker struct {
	authController auth.IAuthController `$inject:""`
}

func (c *WebsocketUpgradeChecker) ShouldUpgradeProtocol(r *http.Request) error {
	token := r.URL.Query().Get("token")
	if token == "" {
		return errors.New("invalid login token")
	}
	panic("idk what to do!?!?!?")
}
