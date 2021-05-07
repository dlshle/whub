package WRCommon

type IHealthCheckExecutor interface {
	DoHealthCheck() error
}

type DefaultHealthCheckExecutor struct {
	senderId string
	receiverId string
	*WRConnection
}

func NewDefaultHealthCheckExecutor(senderId, receiverId string, conn *WRConnection) *DefaultHealthCheckExecutor {
	return &DefaultHealthCheckExecutor{senderId, receiverId, conn}
}

func (e *DefaultHealthCheckExecutor) DoHealthCheck() (err error) {
	_, err = e.Request(NewPingMessage(GenStringId(), e.senderId, e.receiverId))
	return
}
