package connection

import (
	"net/http"
	"whub/common/logger"
)

type IServer interface {
	Start() error
	Stop() error
	OnConnectionError(func(IConnection, error))
	OnClientConnected(func(iConnection IConnection))
	OnClientClosed(func(iConnection IConnection))
	SetLogger(*logger.SimpleLogger)
	OnNonUpgradableRequest(func(http.ResponseWriter, *http.Request))
}
