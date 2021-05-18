package relay_common

import (
	"errors"
	"strings"
	"sync"
	"time"
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
	ServiceCenterRegisterService = ServerServiceCenterUri + "/register" // payload = service descriptor
	ServiceCenterUnregisterService = ServerServiceCenterUri + "/unregister" // payload = service descriptor
	ServiceCenterUpdateService = ServerServiceCenterUri + "/update" // payload = service descriptor
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
		Build()
}

type IServiceCenterClient interface {
	RegisterService(descriptor ServiceDescriptor) error
	UnregisterService(descriptor ServiceDescriptor) error
	UpdateService(descriptor ServiceDescriptor) error
	Response(message *messages.Message) error
	draftDescriptorMessageWith(uri string, descriptor ServiceDescriptor) *messages.Message
	requestMessage(message *messages.Message) error
}

type ServiceCenterClient struct {
	clientCtx *WRContext
	server *WRServer
}

func (c *ServiceCenterClient) draftDescriptorMessageWith(uri string, descriptor ServiceDescriptor) *messages.Message {
	return c.server.DraftMessage(
		c.clientCtx.Identity().Id(),
		c.server.Id(),
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

func (c *ServiceCenterClient) RegisterService(descriptor ServiceDescriptor) (err error) {
	return c.requestMessage(c.draftDescriptorMessageWith(ServiceCenterUnregisterService, descriptor))
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

func (c *ServiceCenterClient) DraftMessage(to string, uri string, msgType int, payload []byte) *messages.Message {
	return c.server.DraftMessage(c.clientCtx.Identity().Id(), to, uri, msgType, payload)
}

type BaseService struct {
	serviceCenterClient IServiceCenterClient
	ctx *WRContext
	id string
	uriPrefix string
	description string
	serviceUris []string
	requestHandlers map[string]messages.SimpleMessageHandler
	hostInfo *RoleDescriptor
	serviceType int
	accessType int
	executionType int
	descriptor *ServiceDescriptor
	cTime time.Time
	lock *sync.RWMutex
}

type IBaseService interface {
	Id() string
	Description() string
	UpdateDescription(string) error
	ServiceUris() []string
	FullServiceUris() []string
	SupportsUri(uri string) bool
	CTime() time.Time
	Handle(*messages.Message) error
	HostInfo() RoleDescriptor
	ServiceType() int
	AccessType() int
	ExecutionType() int
	RegisterMicroService(shortUri string, handler messages.SimpleMessageHandler) error // should update service descriptor to the host
	UnregisterMicroService(shortUri string) error // should update service descriptor to the host
	NotifyHostForUpdate() error
	NewMessage(to string, uri string, msgType int, payload []byte) *messages.Message
	Descriptor() ServiceDescriptor
	// TODO implement this VV
	Register() error
	Start() error
	Stop() error
	Unregister() error
}

func (s *BaseService) withWrite(cb func()) {
	s.lock.Lock()
	defer s.lock.Unlock()
	cb()
}

func (s *BaseService) Id() string {
	return s.id
}

func (s *BaseService) Description() string {
	return s.description
}

func (s *BaseService) ServiceUris() []string {
	return s.serviceUris
}

func (s *BaseService) FullServiceUris() []string {
	fullUris := make([]string, len(s.serviceUris))
	for i, uri := range s.ServiceUris() {
		fullUris[i] = s.uriPrefix + uri
	}
	return fullUris
}

func (s *BaseService) SupportsUri(uri string) bool {
	if !strings.HasPrefix(uri, s.uriPrefix) {
		return false
	}
	actualUri := strings.TrimPrefix(uri, s.uriPrefix)
	for _, v := range s.ServiceUris() {
		if strings.HasPrefix(actualUri, v) {
			return true
		}
	}
	return false
}

func (s *BaseService) CTime() time.Time {
	return s.cTime
}

func (s *BaseService) UpdateDescription(desc string) (err error) {
	err = s.NotifyHostForUpdate()
	if err == nil {
		s.withWrite(func() {
			s.description = desc
		})
	}
	return
}

func (s *BaseService) ServiceType() int {
	return s.serviceType
}

func (s *BaseService) AccessType() int {
	return s.accessType
}

func (s *BaseService) ExecutionType() int {
	return s.executionType
}

func (s *BaseService) HostInfo() RoleDescriptor {
	return *s.hostInfo
}

// returns the corresponding raw uri of the service or ""
func (s *BaseService) matchUri(uri string) (string, error) {
	actualUri := strings.TrimPrefix(uri, s.uriPrefix)
	for _, v := range s.ServiceUris() {
		if strings.HasPrefix(actualUri, v) {
			return v, nil
		}
	}
	return "", errors.New("no matched uri")
}

func (s *BaseService) Handle(message *messages.Message) error {
	matchedUri, err := s.matchUri(message.Uri())
	if err != nil {
		// if no matched uri
		err = s.serviceCenterClient.Response(messages.NewErrorMessage(message.Id(), s.ctx.Identity().Id(), s.HostInfo().Id, message.Uri(), err.Error()))
		return err
	} else {
		// if has matched uri, try to handle it
		msg, err := s.requestHandlers[matchedUri](message)
		if err != nil {
			return err
		} else if msg != nil && msg.MessageType() == messages.MessageTypeError {
			return errors.New((string)(msg.Payload()))
		}
	}
	return nil
}

func (s *BaseService) RegisterMicroService(shortUri string, handler messages.SimpleMessageHandler) error {
	s.withWrite(func() {
		s.serviceUris = append(s.serviceUris, shortUri)
		s.requestHandlers[shortUri] = handler
	})
	return s.NotifyHostForUpdate()
}

func (s *BaseService) UnregisterMicroService(shortUri string) error {
	uriIndex := -1
	for i, uri := range s.ServiceUris() {
		if uri == shortUri {
			uriIndex = i
		}
	}
	if uriIndex == -1 {
		return errors.New("shortUri " + shortUri + " does not exist")
	}
	s.withWrite(func() {
		l := len(s.serviceUris)
		s.serviceUris[l-1], s.serviceUris[uriIndex] = s.serviceUris[uriIndex], s.serviceUris[l-1]
		s.serviceUris = s.serviceUris[:l-1]
		delete(s.requestHandlers, shortUri)
	})
	return s.NotifyHostForUpdate()
}

func (s *BaseService) NotifyHostForUpdate() error {
	if s.serviceCenterClient != nil {
		return s.serviceCenterClient.UpdateService(s.Descriptor())
	}
	return errors.New("no serviceCenterClient found")
}

func (s *BaseService) NewMessage(to string, uri string, msgType int, payload []byte) *messages.Message {
	return s.ctx.Identity().DraftMessage(s.ctx.Identity().Id(), to, uri, msgType, payload)
}

func (s *BaseService) Descriptor() ServiceDescriptor {
	return ServiceDescriptor{
		Id: s.Id(),
		Description: s.Description(),
		HostInfo: s.HostInfo(),
		Provider: s.ctx.Identity().Describe(),
		ServiceUris: s.ServiceUris(),
		CTime: s.CTime(),
		ServiceType: s.ServiceType(),
		AccessType: s.AccessType(),
		ExecutionType: s.ExecutionType(),
	}
}