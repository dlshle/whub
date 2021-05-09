package relay_common

import "wsdk/relay_common/messages"

type IRequestExecutor interface {
	Execute(*messages.ServiceMessage)
}