package mocks

import (
	"wsdk/common/async"
	"wsdk/websocket/wserver"
)

const (
	InitOnGoing = 0
	InitSuccess = 1
	InitFailed  = 2
)

var MockServer *wserver.WServer
var SInitBarrier *async.StatefulBarrier

func StartServer() {
	SInitBarrier = async.NewStatefulBarrier()
	MockServer = wserver.NewWServer(wserver.NewServerConfig("mock", "127.0.0.1", 14719, wserver.DefaultWsConnHandler()))
	err := MockServer.Start()
	if err == nil {
		SInitBarrier.OpenWith(InitSuccess)
	} else {
		SInitBarrier.OpenWith(InitFailed)
		panic("MockInit failed")
	}
}
