package relay_common

import (
	"time"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/utils"
)

const DefaultHealthCheckTimeout = time.Second * 10

type IHealthCheckExecutor interface {
	DoHealthCheck() error
}

type DefaultHealthCheckExecutor struct {
	senderId string
	receiverId string
	timeout time.Duration
	*connection.WRConnection
}

func NewDefaultHealthCheckExecutor(senderId, receiverId string, conn *connection.WRConnection) *DefaultHealthCheckExecutor {
	return &DefaultHealthCheckExecutor{senderId, receiverId, DefaultHealthCheckTimeout, conn}
}

func NewDefaultHealthCheckExecutorWithTimeout(senderId, receiverId string, timeout time.Duration, conn *connection.WRConnection) *DefaultHealthCheckExecutor {
	return &DefaultHealthCheckExecutor{senderId, receiverId, timeout, conn}
}

func (e *DefaultHealthCheckExecutor) DoHealthCheck() (err error) {
	_, err = e.RequestWithTimeout(messages.NewPingMessage(utils.GenStringId(), e.senderId, e.receiverId), e.timeout)
	return
}
