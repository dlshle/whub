package service

import (
	"time"
	"wsdk/common/json"
	"wsdk/relay_common"
)

/*
 * Service can be provided by both client and server.
 * A server service is a collection of local/remote(to client) requests identified by a set of service uris.
 * A client service is a collection of function calls identified by a set of service uris.
 * Client service anatomy:
 *  Service { handlers[path]handler }
 * Usual Service Flow:
 *  ClientX -request-> Server -request-> serverServiceHandler -request-> Client -request-> clientServiceHandler -response-> Server -response-> ClientX
 */

const (
	ServicePrefix = "/service"
)

const (
	ServerServiceCenterUri = ServicePrefix + "/center"
)

// ServerServiceCenter Micro-services
const (
	ServiceCenterRegisterService   = ServerServiceCenterUri + "/register"   // payload = service descriptor
	ServiceCenterUnregisterService = ServerServiceCenterUri + "/unregister" // payload = service descriptor
	ServiceCenterUpdateService     = ServerServiceCenterUri + "/update"     // payload = service descriptor
)

type ServiceDescriptor struct {
	Id            string
	Description   string
	HostInfo      relay_common.RoleDescriptor // server id
	Provider      relay_common.RoleDescriptor
	ServiceUris   []string
	CTime         time.Time
	ServiceType   int
	AccessType    int
	ExecutionType int
	Status        int
}

func (sd ServiceDescriptor) String() string {
	return json.NewJsonBuilder().
		Put("id", sd.Id).
		Put("description", sd.Description).
		Put("hostInfo", sd.HostInfo.String()).
		Put("provider", sd.Provider.String()).
		Put("serviceUris", json.BracketStrings(sd.ServiceUris)).
		Put("cTime", sd.CTime.String()).
		Put("serviceType", (string)(sd.ServiceType)).
		Put("accessType", (string)(sd.AccessType)).
		Put("executionType", (string)(sd.ExecutionType)).
		Put("status", (string)(sd.Status)).
		Build()
}

const (
	ServiceStatusUnregistered = 0
	ServiceStatusIdle         = 1 // for server only
	ServiceStatusRegistered   = 1 // for client only
	ServiceStatusStarting     = 2
	ServiceStatusRunning      = 3
	ServiceStatusBlocked      = 4 // when queue is maxed out
	ServiceStatusDead         = 5 // health check fails
	ServiceStatusStopping     = 6
)

type IBaseService interface {
	Id() string
	Description() string
	ServiceUris() []string
	FullServiceUris() []string
	SupportsUri(uri string) bool
	CTime() time.Time
	ServiceType() int
	AccessType() int
	ExecutionType() int

	HealthCheck() error

	Start() error
	Stop() error
	Status() int

	Cancel(messageId string) error

	KillAllProcessingJobs() error
	CancelAllPendingJobs() error

	Describe() ServiceDescriptor
}
