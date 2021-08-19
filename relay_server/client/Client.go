package client

import (
	"wsdk/relay_common/roles"
)

type Client struct {
	*roles.CommonClient
}

func NewClient(id string, description string, cType int, cKey string, pScope int) *Client {
	return &Client{roles.NewClient(id, description, cType, cKey, pScope)}
}

func NewClientFromDescriptor(descriptor roles.RoleDescriptor, infoDescriptor roles.ClientExtraInfoDescriptor) *Client {
	return &Client{roles.NewClientByDescriptor(descriptor, infoDescriptor)}
}
