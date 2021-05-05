package WRClient

import (
	"sync"
	WSClient "wsdk/WClient"
	"wsdk/WRCommon"
)

type WRClient struct {
	c *WSClient.WClient
	requestMap map[string][]func() // id -- [listener functions]
	l *sync.RWMutex
}

type IWRClient interface {
	Request(message *WRCommon.Message) (*WRCommon.Message, error)
	Send([]byte) error
	OnMessage(string, func())
	OffMessage(string, func())
	OffAllMessage(string)
}