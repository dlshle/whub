package relay_common

import (
	"wsdk/relay_common/service"
)

type IRequestExecutor interface {
	Execute(*service.ServiceRequest)
}
