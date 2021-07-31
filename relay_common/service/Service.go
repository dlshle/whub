package service

import (
	"fmt"
	"strings"
	"time"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/roles"
)

/*
 * Service can be provided by both client_manager and server.
 * A server service_manager is a collection of local/remote(to client_manager) requests identified by a set of service_manager uris.
 * A client_manager service_manager is a collection of function calls identified by a set of service_manager uris.
 * Client service_manager anatomy:
 *  Service { handlers(deprecated)[path]handler }
 * Usual Service Flow:
 *  ClientX -request-> Server -request-> servicePool -request-> serverRequestExecutor -request-> Client -request-> servicePool -request-> clientRequestExecutor -request-> clientServiceHandler -response-> Server -response-> ClientX
 */

func init() {
	initServiceStatusStrMap()
}

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

func (sd ServiceDescriptor) marshallStringField(key string, value string) string {
	return fmt.Sprintf("\"%s\":\"%s\"", key, value)
}

func (sd ServiceDescriptor) marshallObjField(key string, value string) string {
	return fmt.Sprintf("\"%s\":%s", key, value)
}

func (sd ServiceDescriptor) marshallArrStringField(key string, values []string) string {
	l := len(values)
	var builder strings.Builder
	builder.WriteByte('"')
	builder.WriteString(key)
	builder.WriteString("\":[")
	for i, v := range values {
		builder.WriteString(fmt.Sprintf("\"%s\"", v))
		if i != l-1 {
			builder.WriteByte(',')
		}
	}
	builder.WriteByte(']')
	return builder.String()
}

func (sd ServiceDescriptor) marshallNumberField(key string, value int) string {
	return fmt.Sprintf("\"%s\":%d", key, value)
}

func (sd ServiceDescriptor) String() string {
	return fmt.Sprintf("{%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s}",
		sd.marshallStringField("id", sd.Id),
		sd.marshallStringField("description", sd.Description),
		sd.marshallObjField("hostInfo", sd.HostInfo.String()),
		sd.marshallObjField("provider", sd.Provider.String()),
		sd.marshallStringField("description", sd.Description),
		sd.marshallArrStringField("serviceUris", sd.ServiceUris),
		sd.marshallStringField("cTime", sd.CTime.Format("2006-01-02T15:04:05Z07:00")),
		sd.marshallNumberField("serviceType", sd.ServiceType),
		sd.marshallNumberField("accessType", sd.AccessType),
		sd.marshallNumberField("executionType", sd.ExecutionType),
		sd.marshallNumberField("status", sd.Status),
	)
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

var ServiceStatusStringMap map[int]string

func initServiceStatusStrMap() {
	ServiceStatusStringMap = make(map[int]string)
	ServiceStatusStringMap[ServiceStatusUnregistered] = "unregistered"
	ServiceStatusStringMap[ServiceStatusIdle] = "idle"
	ServiceStatusStringMap[ServiceStatusRegistered] = "registered"
	ServiceStatusStringMap[ServiceStatusStarting] = "starting"
	ServiceStatusStringMap[ServiceStatusRunning] = "running"
	ServiceStatusStringMap[ServiceStatusBlocked] = "blocked"
	ServiceStatusStringMap[ServiceStatusDead] = "dead"
	ServiceStatusStringMap[ServiceStatusStopping] = "stopping"
}

// Service Uri should always be /service_manager/serviceId/uri_trie/params

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
