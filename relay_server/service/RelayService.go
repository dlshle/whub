package service

import (
	"fmt"
	"wsdk/relay_common"
	"wsdk/relay_common/service"
	"wsdk/relay_server"
)

type RelayService struct {
	*ServerService
}

type IRelayService interface {
	IServerService
	RestoreExternally(reconnectedOwner *relay_server.WRServerClient) error
	Update(descriptor service.ServiceDescriptor) error
}

func NewRelayService(ctx relay_common.IWRContext,
	descriptor service.ServiceDescriptor,
	provider IServiceProvider,
	executor relay_common.IRequestExecutor) IServerService {
	return &RelayService{
		NewService(ctx, descriptor.Id, descriptor.Description, provider, executor, descriptor.ServiceUris, descriptor.ServiceType, descriptor.AccessType, descriptor.ExecutionType),
	}
}

func (s *RelayService) RestoreExternally(reconnectedOwner *relay_server.WRServerClient) (err error) {
	if s.Status() != service.ServiceStatusDead {
		err = NewInvalidServiceStatusError(s.Id(), s.Status(), fmt.Sprintf(" status should be %d to be restored externally", service.ServiceStatusDead))
		return
	}
	if err = s.Stop(); err != nil {
		return
	}
	oldOwner := s.provider
	oldPool := s.serviceQueue
	s.withWrite(func() {
		s.provider = reconnectedOwner
		s.serviceQueue = service.NewServiceTaskQueue(reconnectedOwner.MessageRelayExecutor(), s.ctx.ServiceTaskPool())
	})
	err = s.Start()
	if err != nil {
		// fallback to previous status
		s.withWrite(func() {
			s.provider = oldOwner
			s.serviceQueue = oldPool
			s.status = service.ServiceStatusDead
		})
	}
	return err
}

func (s *ServerService) Update(descriptor service.ServiceDescriptor) (err error) {
	oldDescriptor := s.Describe()
	s.update(descriptor)
	if descriptor.Status == service.ServiceStatusStarting {
		err = s.Start()
		if err != nil {
			s.update(oldDescriptor)
		}
	}
	return nil
}

func (s *ServerService) update(descriptor service.ServiceDescriptor) {
	s.withWrite(func() {
		s.description = descriptor.Description
		s.status = descriptor.Status
		s.serviceType = descriptor.ServiceType
		s.executionType = descriptor.ExecutionType
		s.cTime = descriptor.CTime
		s.serviceUris = descriptor.ServiceUris
	})
}
