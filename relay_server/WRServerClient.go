package relay_server

import (
	"wsdk/relay_common"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/utils"
)

type WRServerClient struct {
	*relay_common.WRClient
	requestExecutor     relay_common.IRequestExecutor
	healthCheckExecutor relay_common.IHealthCheckExecutor
}

func (c *WRServerClient) RequestExecutor() relay_common.IRequestExecutor {
	return c.requestExecutor
}

func (c *WRServerClient) HealthCheckExecutor() relay_common.IHealthCheckExecutor {
	return c.healthCheckExecutor
}

// since client is to server, so the drafted messages is to the client
func (c *WRServerClient) DraftMessage(from string, uri string, msgType int, payload []byte) *messages.Message {
	return messages.NewMessage(utils.GenStringId(), from, c.Id(), uri, msgType, payload)
}

func NewAnonymousClient(ctx *relay_common.WRContext, conn *connection.WRConnection) *WRServerClient {
	return NewClient(ctx, conn, "_", "", relay_common.ClientTypeAnonymous, "", relay_common.PRMessage)
}

func NewClient(ctx *relay_common.WRContext, conn *connection.WRConnection, id string, description string, cType int, cKey string, pScope int) *WRServerClient {
	return &WRServerClient{relay_common.NewClient(conn, id, description, cType, cKey, pScope), messages.NewServiceMessageExecutor(conn), relay_common.NewDefaultHealthCheckExecutor(ctx.Identity().Id(), id, conn)}
}
