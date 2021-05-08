package WRCommon

import (
	"wsdk/WRCommon/Connection"
	"wsdk/WRCommon/Message"
	"wsdk/WRCommon/Utils"
)

type IHealthCheckExecutor interface {
	DoHealthCheck() error
}

type DefaultHealthCheckExecutor struct {
	senderId string
	receiverId string
	*Connection.WRConnection
}

func NewDefaultHealthCheckExecutor(senderId, receiverId string, conn *Connection.WRConnection) *DefaultHealthCheckExecutor {
	return &DefaultHealthCheckExecutor{senderId, receiverId, conn}
}

func (e *DefaultHealthCheckExecutor) DoHealthCheck() (err error) {
	_, err = e.Request(Message.NewPingMessage(Utils.GenStringId(), e.senderId, e.receiverId))
	return
}
