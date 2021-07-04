package service

import (
	"encoding/json"
	"time"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/roles"
)

/*
 * Service can be provided by both client and server.
 * A server service is a collection of local/remote(to client) requests identified by a set of service uris.
 * A client service is a collection of function calls identified by a set of service uris.
 * Client service anatomy:
 *  Service { handlers(deprecated)[path]handler }
 * Usual Service Flow:
 *  ClientX -request-> Server -request-> servicePool -request-> serverRequestExecutor -request-> Client -request-> servicePool -request-> clientRequestExecutor -request-> clientServiceHandler -response-> Server -response-> ClientX
 */

const (
	ServicePrefix = "/service"
)

type ServiceDescriptor struct {
	Id            string               `json:"id"`
	Description   string               `json:"description"`
	HostInfo      roles.RoleDescriptor `json:"hostInfo"`
	Provider      roles.RoleDescriptor `json:"provider"`
	ServiceUris   []string             `json:"serviceUris"`
	CTime         time.Time            `json:"cTime"`
	ServiceType   int                  `json:"serviceType"`
	AccessType    int                  `json:"accessType"`
	ExecutionType int                  `json:"executionType"`
	Status        int                  `json:"status"`
}

func (sd ServiceDescriptor) String() string {
	marshalled, err := json.Marshal(sd)
	if err != nil {
		return "error"
	}
	return (string)(marshalled)
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

// Service Uri should always be /service/serviceId/uri/params

// Service access type
const (
	ServiceAccessTypeHttp   = 0
	ServiceAccessTypeSocket = 1
	ServiceAccessTypeBoth   = 2
)

// Service execution type
const (
	ServiceExecutionAsync = 0
	ServiceExecutionSync  = 1
)

// Service type
const (
	ServiceTypeInternal = 0
	ServiceTypeRelay    = 1
)

type RequestHandler func(request *ServiceRequest, pathParams map[string]string, queryParams map[string]string) error

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

	Start() error
	Stop() error
	Status() int

	OnStarted(func(IBaseService))
	OnStopped(func(IBaseService))

	Handle(message *messages.Message) *messages.Message

	Cancel(messageId string) error

	KillAllProcessingJobs() error
	CancelAllPendingJobs() error

	ProviderInfo() roles.RoleDescriptor
	HostInfo() roles.RoleDescriptor

	Describe() ServiceDescriptor
}
