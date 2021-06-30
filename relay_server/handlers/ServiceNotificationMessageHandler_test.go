package handlers

import (
	"testing"
	"time"
	"wsdk/common/test_utils"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/roles"
	"wsdk/relay_common/service"
	"wsdk/relay_server/context"
)

func TestServiceNotificationMessageHandler(t *testing.T) {
	mockConn := connection.NewMockConnection()
	mockServer := roles.NewServer("s", "xx", "localhost", 1234)
	mockClient := roles.NewClient(mockConn, "c", "x", 0, "k", 1)
	context.Ctx.Start(mockServer)
	h := NewServiceNotificationMessageHandler()
	sd := service.ServiceDescriptor{
		Id:            "s",
		Description:   "x",
		HostInfo:      mockServer.Describe(),
		Provider:      mockClient.Describe(),
		ServiceUris:   []string{"/x/y", "/y/:z"},
		CTime:         time.Now(),
		ServiceType:   1,
		AccessType:    2,
		ExecutionType: 3,
		Status:        service.ServiceStatusRunning,
	}
	test_utils.NewTestGroup("ServiceNotificationMessageHandler", "").Cases([]*test_utils.Assertion{
		test_utils.NewTestCase("handle msg w/o payload", "should return error", func() bool {
			err := h.Handle(messages.NewMessage("123", "c", "s", "a/b/c", messages.MessageTypeClientServiceNotification, []byte{}), mockConn)
			return err != nil
		}),
		test_utils.NewTestCase("handle msg w/o payload", "should return error", func() bool {
			err := h.Handle(messages.NewMessage("123", "c", "s", "a/b/c", messages.MessageTypeClientServiceNotification, ([]byte)(sd.String())), mockConn)
			return err != nil
		}),
	}).Do(t)
}
