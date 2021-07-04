package client

import (
	"wsdk/relay_common/connection"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/roles"
	"wsdk/relay_common/service"
	"wsdk/relay_common/utils"
	server_service "wsdk/relay_server/service"
)

type Client struct {
	*roles.CommonClient
	messageRelayExecutor service.IRequestExecutor
}

func (c *Client) MessageRelayExecutor() service.IRequestExecutor {
	return c.messageRelayExecutor
}

func (c *Client) NewMessage(from string, uri string, msgType int, payload []byte) *messages.Message {
	return messages.NewMessage(utils.GenStringId(), from, c.Id(), uri, msgType, payload)
}

func NewAnonymousClient(conn connection.IConnection) *Client {
	return NewClient(conn, conn.Address(), "", roles.ClientTypeAnonymous, "", roles.PRMessage)
}

func NewClient(conn connection.IConnection, id string, description string, cType int, cKey string, pScope int) *Client {
	return &Client{roles.NewClient(conn, id, description, cType, cKey, pScope),
		server_service.NewRelayServiceRequestExecutor(conn),
	}
}
