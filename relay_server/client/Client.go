package client

import (
	"wsdk/relay_common/connection"
	"wsdk/relay_common/health_check"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/roles"
	"wsdk/relay_common/service"
	"wsdk/relay_common/utils"
	"wsdk/relay_server/context"
	"wsdk/relay_server/request"
)

type Client struct {
	*roles.CommonClient
	messageRelayExecutor service.IRequestExecutor
	healthCheckExecutor  health_check.IHealthCheckExecutor
}

func (c *Client) MessageRelayExecutor() service.IRequestExecutor {
	return c.messageRelayExecutor
}

func (c *Client) HealthCheckExecutor() health_check.IHealthCheckExecutor {
	return c.healthCheckExecutor
}

// since client is to server, so the drafted messages is to the client
func (c *Client) DraftMessage(from string, to string, uri string, msgType int, payload []byte) *messages.Message {
	return messages.NewMessage(utils.GenStringId(), from, c.Id(), uri, msgType, payload)
}

func (c *Client) NewMessage(from string, uri string, msgType int, payload []byte) *messages.Message {
	return messages.NewMessage(utils.GenStringId(), from, c.Id(), uri, msgType, payload)
}

func NewAnonymousClient(ctx *context.Context, conn *connection.Connection) *Client {
	return NewClient(ctx, conn, "_", "", roles.ClientTypeAnonymous, "", roles.PRMessage)
}

func NewClient(ctx *context.Context, conn *connection.Connection, id string, description string, cType int, cKey string, pScope int) *Client {
	return &Client{roles.NewClient(conn, id, description, cType, cKey, pScope), request.RelayRequestExecutor(ctx, conn), health_check.NewDefaultHealthCheckExecutor(ctx.Server().Id(), id, conn)}
}
