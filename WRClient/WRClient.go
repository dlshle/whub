package WRClient

import (
	"sync"
	"wsdk/WRCommon/Message"
	WSClient2 "wsdk/base/WClient"
)

type WRClient struct {
	c *WSClient2.WClient
	requestMap map[string][]func() // id -- [listener functions]
	l *sync.RWMutex
}

type IWRClient interface {
	Request(message *Message.Message) (*Message.Message, error)
	Send([]byte) error
	OnMessage(string, func())
	OffMessage(string, func())
	OffAllMessage(string)
}