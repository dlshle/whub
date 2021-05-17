package relay_common

import "time"

const (
	ServicePrefix = "/service"
)

type ServiceDescriptor struct {
	Id            string
	Description   string
	HostInfo      RoleDescriptor // server id
	Provider      RoleDescriptor
	ServiceUris   []string
	CTime         time.Time
	ServiceType   int
	AccessType    int
	ExecutionType int
}


