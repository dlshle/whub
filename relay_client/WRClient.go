package relay_client

import (
	"sync"
	wclient "wsdk/base/wclient"
	"wsdk/relay_common/messages"
)

// TODO
type WRClient struct {
	c          *wclient.WClient
	serviceMap map[string]IClientService // id -- [listener functions]
	l          *sync.RWMutex
}

type IWRClient interface {
	Request(message *messages.Message) (*messages.Message, error)
	Send([]byte) error
	OnMessage(string, func())
	OffMessage(string, func())
	OffAllMessage(string)
}
