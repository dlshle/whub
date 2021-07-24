package messaging

import (
	"testing"
	"wsdk/common/test_utils"
	"wsdk/common/utils"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/roles"
	"wsdk/relay_server/context"
)

func TestMessagingService(t *testing.T) {
	context.Ctx.Start(roles.NewServer("123", "asd", "qwe", 123))
	service := New()
	test_utils.NewTestGroup("MessagingService", "").Cases([]*test_utils.Assertion{
		test_utils.NewTestCase("test service_manager uris", "", func() bool {
			uris := utils.StringArrayToInterfaceArray(service.ServiceUris())
			t.Log("uris: ", uris)
			expectedUris := utils.StringArrayToInterfaceArray([]string{RouteSend, RouteBroadcast})
			return test_utils.AssertSlicesEqual(uris, expectedUris)
		}),
		test_utils.NewTestCase("test routes", "", func() bool {
			msg := service.Handle(messages.NewMessage("x", "client", "server", "/service_manager/messaging/send", messages.MessageTypeServiceRequest, ([]byte)("asdasd")))
			t.Log("msg: ", msg)
			return msg.MessageType() == messages.MessageTypeError
		}),
		test_utils.NewTestCase("test ioc", "", func() bool {
			ms := service.(*MessagingService)
			return ms.IClientManager != nil
		}),
	}).Do(t)
}
