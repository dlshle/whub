package relay_client

import (
	"wsdk/relay_common/service"
)

type ClientServiceExecutor struct {
	mManager service.IServiceHandler
}

func (e *ClientServiceExecutor) Execute(*service.ServiceRequest) {

}
