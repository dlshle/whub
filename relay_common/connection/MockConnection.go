package connection

import (
	"time"
	"wsdk/common/async"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/notification"
)

type MockConnection struct {
}

func (m *MockConnection) Address() string {
	return ""
}

func (m *MockConnection) AsyncRequest(msg *messages.Message) (*async.StatefulBarrier, error) {
	return nil, nil
}

func (m *MockConnection) Request(message *messages.Message) (*messages.Message, error) {
	return nil, nil
}

func (m *MockConnection) RequestWithTimeout(message *messages.Message, duration time.Duration) (*messages.Message, error) {
	return nil, nil
}

func (m *MockConnection) Send(message *messages.Message) error {
	return nil
}

func (m *MockConnection) OnIncomingMessage(f func(message *messages.Message)) {
}

func (m *MockConnection) OnceMessage(s string, f func(*messages.Message)) (notification.Disposable, error) {
	return nil, nil
}

func (m *MockConnection) OnMessage(s string, f func(*messages.Message)) (notification.Disposable, error) {
	return nil, nil
}

func (m *MockConnection) OffMessage(s string, f func(*messages.Message)) {

}

func (m *MockConnection) OffAll(s string) {

}

func (m *MockConnection) OnError(f func(error)) {

}

func (m *MockConnection) OnClose(f func(error)) {

}

func (m *MockConnection) Close() error {
	return nil
}

func NewMockConnection() IConnection {
	return &MockConnection{}
}
