package relay_client

import (
	"sync"
	WSClient2 "wsdk/base/wclient"
	"wsdk/relay_common/messages"
)

// TODO
type WRClient struct {
	c *WSClient2.WClient
	serviceMap map[string]IClientService // id -- [listener functions]
	l *sync.RWMutex
}

type IWRClient interface {
	Request(message *messages.Message) (*messages.Message, error)
	Send([]byte) error
	OnMessage(string, func())
	OffMessage(string, func())
	OffAllMessage(string)
}