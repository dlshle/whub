package relay_server

import (
	"encoding/json"
	"wsdk/relay_common/service"
)

func ParseServiceDescriptor(payload []byte) (*service.ServiceDescriptor, error) {
	var descriptor *service.ServiceDescriptor
	err := json.Unmarshal(payload, descriptor)
	if err != nil {
		return nil, err
	}
	return descriptor, nil
}
