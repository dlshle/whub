package relay_client

import (
	"wsdk/relay_common/messages"
	"wsdk/relay_common/service"
)

type ClientServiceExecutor struct {
	mManager service.IMicroServiceManager
}

func (e *ClientServiceExecutor) Execute(*messages.ServiceRequest) {

}