package WRelayServer

import "wsdk/WRCommon"

type WRServerClient struct {
	*WRCommon.WRClient
	requestExecutor WRCommon.IRequestExecutor
	healthCheckExecutor WRCommon.IHealthCheckExecutor
}

type IWRServerClient interface {
	WRCommon.IWRClient
	RequestExecutor() WRCommon.IRequestExecutor
	HealthCheckExecutor() WRCommon.IHealthCheckExecutor
	DraftMessage(to string, msgType int, payload []byte) *WRCommon.Message
}

func (c *WRServerClient) RequestExecutor() WRCommon.IRequestExecutor {
	return c.requestExecutor
}

func (c *WRServerClient) HealthCheckExecutor() WRCommon.IHealthCheckExecutor {
	return c.healthCheckExecutor
}

// since client is to server, so the drafted message is to the client
func (c *WRServerClient) DraftMessage(from string, msgType int, payload []byte) *WRCommon.Message {
	return WRCommon.NewMessage(WRCommon.GenStringId(), from, c.Id(), msgType, payload)
}

func NewAnonymousClient(conn *WRCommon.WRConnection) *WRServerClient {
	return NewClient(conn, "", "", WRCommon.ClientTypeAnonymous, "", WRCommon.PRMessage)
}

func NewClient(conn *WRCommon.WRConnection, id string, description string, cType int, cKey string, pScope int) *WRServerClient {
	// TODO WRCommon.NewDefaultHealthCheckExecutor is incorrect, we need to specify the correct senderId from context!!!
	return &WRServerClient{WRCommon.NewClient(conn, id, description, cType, cKey, pScope), WRCommon.NewServiceMessageExecutor(conn), WRCommon.NewDefaultHealthCheckExecutor(id, id, conn)}
}
