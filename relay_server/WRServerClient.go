package relay_server

import (
	"wsdk/relay_common"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/utils"
	"wsdk/relay_server/service"
)

type WRServerClient struct {
	*relay_common.WRClient
	messageRelayExecutor relay_common.IRequestExecutor
	healthCheckExecutor  relay_common.IHealthCheckExecutor
}

func (c *WRServerClient) MessageRelayExecutor() relay_common.IRequestExecutor {
	return c.messageRelayExecutor
}

func (c *WRServerClient) HealthCheckExecutor() relay_common.IHealthCheckExecutor {
	return c.healthCheckExecutor
}

// since client is to server, so the drafted messages is to the client
func (c *WRServerClient) DraftMessage(from string, to string, uri string, msgType int, payload []byte) *messages.Message {
	return messages.NewMessage(utils.GenStringId(), from, c.Id(), uri, msgType, payload)
}

func (c *WRServerClient) NewMessage(from string, uri string, msgType int, payload []byte) *messages.Message {
	return messages.NewMessage(utils.GenStringId(), from, c.Id(), uri, msgType, payload)
}

func NewAnonymousClient(ctx *Context, conn *connection.WRConnection) *WRServerClient {
	return NewClient(ctx, conn, "_", "", relay_common.ClientTypeAnonymous, "", relay_common.PRMessage)
}

func NewClient(ctx *Context, conn *connection.WRConnection, id string, description string, cType int, cKey string, pScope int) *WRServerClient {
	return &WRServerClient{relay_common.NewClient(conn, id, description, cType, cKey, pScope), service.RelayRequestExecutor(ctx, conn), relay_common.NewDefaultHealthCheckExecutor(ctx.Identity().Id(), id, conn)}
}
