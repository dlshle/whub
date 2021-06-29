package messaging

import (
	"testing"
	"wsdk/common/test_utils"
	"wsdk/common/utils"
	"wsdk/relay_common/messages"
	"wsdk/relay_server/context"
	"wsdk/relay_server/managers"
)

func TestMessagingService(t *testing.T) {
	ctx := context.MockCtx
	service := New(managers.NewClientManager(ctx))
	test_utils.NewTestGroup("MessagingService", "").Cases([]*test_utils.Assertion{
		test_utils.NewTestCase("test service uris", "", func() bool {
			uris := utils.StringArrayToInterfaceArray(service.ServiceUris())
			t.Log("uris: ", uris)
			expectedUris := utils.StringArrayToInterfaceArray([]string{RouteSend, RouteBroadcast})
			return test_utils.AssertSlicesEqual(uris, expectedUris)
		}),
		test_utils.NewTestCase("test routes", "", func() bool {
			msg := service.Handle(messages.NewMessage("x", "client", "server", "/service/messaging/send", messages.MessageTypeServiceRequest, ([]byte)("asdasd")))
			t.Log("msg: ", msg)
			return msg.MessageType() == messages.MessageTypeError
		}),
	}).Do(t)
}
