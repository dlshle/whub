package service

import (
	"fmt"
	"wsdk/relay_common/service"
	"wsdk/relay_server/client"
	"wsdk/relay_server/context"
)

type RelayService struct {
	*Service
}

type IRelayService interface {
	IService
	RestoreExternally(reconnectedOwner *client.Client) error
	Update(descriptor service.ServiceDescriptor) error
}

func NewRelayService(ctx *context.Context,
	descriptor service.ServiceDescriptor,
	provider IServiceProvider,
	executor service.IRequestExecutor) IService {
	return &RelayService{
		NewService(ctx, descriptor.Id, descriptor.Description, provider, executor, descriptor.ServiceUris, descriptor.ServiceType, descriptor.AccessType, descriptor.ExecutionType),
	}
}

func (s *RelayService) RestoreExternally(reconnectedOwner *client.Client) (err error) {
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

func (s *Service) Update(descriptor service.ServiceDescriptor) (err error) {
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

func (s *Service) update(descriptor service.ServiceDescriptor) {
	s.withWrite(func() {
		s.description = descriptor.Description
		s.status = descriptor.Status
		s.serviceType = descriptor.ServiceType
		s.executionType = descriptor.ExecutionType
		s.cTime = descriptor.CTime
		s.serviceUris = descriptor.ServiceUris
	})
}
