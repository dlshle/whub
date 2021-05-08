package WRCommon

import "wsdk/WRCommon/Message"

type IRequestExecutor interface {
	Execute(*Message.ServiceMessage)
}