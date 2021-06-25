package relay_client

import (
	"sync"
	"wsdk/relay_common/messages"
	wclient "wsdk/websocket/wclient"
)

// TODO
type WRClient struct {
	c *wclient.WClient
	// serviceMap map[string]IClientService // id -- [listener functions]
	service IClientService
	server  *Server
	l       *sync.RWMutex
}

type IWRClient interface {
	Request(message *messages.Message) (*messages.Message, error)
	Send([]byte) error
	OnMessage(string, func())
	OffMessage(string, func())
	OffAllMessage(string)
}
