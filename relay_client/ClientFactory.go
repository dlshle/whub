package relay_client

import "wsdk/common/connection"

var clientFactoryMap map[uint8]connection.IClient

func NewClientBy(connType uint8) connection.IClient {
	if clientFactoryMap == nil {
		// TODO
	}
	return nil
}

func initClientFactoryMap() {
	clientFactoryMap = make(map[uint8]connection.IClient)
	// clientFactoryMap[connection.TypeTCP] =
	// TODO
}
