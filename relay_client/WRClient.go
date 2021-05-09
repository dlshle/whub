package relay_client

import (
	"sync"
	"wsdk/relay_common/messages"
	WSClient2 "wsdk/base/wclient"
)

type WRClient struct {
	c *WSClient2.WClient
	requestMap map[string][]func() // id -- [listener functions]
	l *sync.RWMutex
}

type IWRClient interface {
	Request(message *messages.Message) (*messages.Message, error)
	Send([]byte) error
	OnMessage(string, func())
	OffMessage(string, func())
	OffAllMessage(string)
}