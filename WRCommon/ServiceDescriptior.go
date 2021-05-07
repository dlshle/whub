package WRCommon

import "time"

type ServiceDescriptor struct {
	Id                  string
	Description         string
	HostInfo			RoleDescriptor // server id
	Owner               IWRClient
	ServiceUris         []string
	CTime               time.Time
	ServiceType         int
	AccessType          int
	ExecutionType       int
}


