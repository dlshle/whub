package service

import (
	"errors"
	"time"
	"wsdk/relay_common"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/utils"
)

const (
	ServicePrefix = "/service"
)

const (
	ServerServiceCenterUri = ServicePrefix + "/center"
)

// ServerServiceCenter Micro-services
const (
	ServiceCenterRegisterService = ServerServiceCenterUri + "/register"     // payload = service descriptor
	ServiceCenterUnregisterService = ServerServiceCenterUri + "/unregister" // payload = service descriptor
	ServiceCenterUpdateService = ServerServiceCenterUri + "/update"         // payload = service descriptor
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
	return utils.NewJsonBuilder().
		Put("id", sd.Id).
		Put("description", sd.Description).
		Put("hostInfo", sd.HostInfo.String()).
		Put("provider", sd.Provider.String()).
		Put("serviceUris", utils.BracketStrings(sd.ServiceUris)).
		Put("cTime", sd.CTime.String()).
		Put("serviceType", (string)(sd.ServiceType)).
		Put("accessType", (string)(sd.AccessType)).
		Put("executionType", (string)(sd.ExecutionType)).
		Put("status", (string)(sd.Status)).
		Build()
}

type IServiceCenterClient interface {
	RegisterService(descriptor ServiceDescriptor) error
	UnregisterService(descriptor ServiceDescriptor) error
	UpdateService(descriptor ServiceDescriptor) error
	Response(message *messages.Message) error
	HealthCheck() error
}

type ServiceCenterClient struct {
	clientCtx *relay_common.WRContext
	server *relay_common.WRServer
}

func (c *ServiceCenterClient) draftDescriptorMessageWith(uri string, descriptor ServiceDescriptor) *messages.Message {
	return c.draftMessage(
		uri,
		messages.MessageTypeClientNotification, ([]byte)(descriptor.String()),
	)
}

func (c *ServiceCenterClient) requestMessage(message *messages.Message) (err error) {
	resp, err := c.server.Request(message)
	if resp != nil && resp.MessageType() == messages.MessageTypeError {
		return errors.New((string)(resp.Payload()))
	}
	return
}

func (c *ServiceCenterClient) HealthCheck() error {
	return c.requestMessage(c.draftMessage("", messages.MessageTypePing, nil))
}

func (c *ServiceCenterClient) RegisterService(descriptor ServiceDescriptor) (err error) {
	return c.requestMessage(c.draftDescriptorMessageWith(ServiceCenterRegisterService, descriptor))
}

func (c *ServiceCenterClient) UnregisterService(descriptor ServiceDescriptor) (err error) {
	return c.requestMessage(c.draftDescriptorMessageWith(ServiceCenterUnregisterService, descriptor))
}

func (c *ServiceCenterClient) UpdateService(descriptor ServiceDescriptor) error {
	return c.requestMessage(c.draftDescriptorMessageWith(ServiceCenterUpdateService, descriptor))
}

func (c *ServiceCenterClient) Response(message *messages.Message) error {
	return c.server.Send(message)
}

func (c *ServiceCenterClient) draftMessage(uri string, msgType int, payload []byte) *messages.Message {
	return c.server.DraftMessage(c.clientCtx.Identity().Id(), c.server.Id(), uri, msgType, payload)
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
