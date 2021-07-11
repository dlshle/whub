package mocks

import (
	"wsdk/websocket/wclient"
)

var MockClient *WSClient.WClient

func StartClient() {
	SInitBarrier.Wait()
	MockClient = WSClient.NewClient("127.0.0.1:14719")
	err := MockClient.Connect()
	if err != nil {
		panic("MockClient init Failed")
	}
}
