package utils

import (
	"encoding/json"
	"whub/hub_common/service"
)

func ParseServiceDescriptor(payload []byte) (service.ServiceDescriptor, error) {
	var descriptor service.ServiceDescriptor
	err := json.Unmarshal(payload, &descriptor)
	if err != nil {
		return descriptor, err
	}
	return descriptor, nil
}
