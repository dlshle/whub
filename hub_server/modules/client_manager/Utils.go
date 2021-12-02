package client_manager

import (
	"encoding/json"
	"whub/common/utils"
	"whub/hub_common/messages"
	"whub/hub_common/roles"
)

func UnmarshallClientDescriptor(message messages.IMessage) (roleDescriptor roles.RoleDescriptor, extraInfoDescriptor roles.ClientExtraInfoDescriptor, err error) {
	err = utils.ProcessWithError([]func() error{
		func() error {
			return json.Unmarshal(message.Payload(), &roleDescriptor)
		},
		func() error {
			return json.Unmarshal(([]byte)(roleDescriptor.ExtraInfo), &extraInfoDescriptor)
		},
	})
	return
}
