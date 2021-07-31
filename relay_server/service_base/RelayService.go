package service_base

import (
	"fmt"
	"wsdk/common/utils"
	"wsdk/relay_common/service"
	"wsdk/relay_server/client"
)

type RelayService struct {
	*Service
}

type IRelayService interface {
	IService
	Init(descriptor service.ServiceDescriptor,
		provider IServiceProvider,
		executor service.IRequestExecutor)
	RestoreExternally(reconnectedOwner *client.Client) error
	Update(descriptor service.ServiceDescriptor) error
}

func (s *RelayService) Init(descriptor service.ServiceDescriptor,
	provider IServiceProvider,
	executor service.IRequestExecutor) {
	s.Service = NewService(descriptor.Id, descriptor.Description, provider, executor, descriptor.ServiceUris, descriptor.ServiceType, descriptor.AccessType, descriptor.ExecutionType)
}

func (s *RelayService) RestoreExternally(reconnectedOwner *client.Client) (err error) {
	defer s.Logger().Println("restoring result: ", utils.ConditionalPick(err != nil, err, "success"))
	s.Logger().Println("restoring externally...")
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
		s.serviceQueue = service.NewServiceTaskQueue(s.HostInfo().Id, reconnectedOwner.MessageRelayExecutor(), s.ctx.ServiceTaskPool())
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

func (s *RelayService) Update(descriptor service.ServiceDescriptor) (err error) {
	defer s.Logger().Println("update result: ", utils.ConditionalPick(err != nil, err, "success"))
	s.Logger().Println("update with descriptor: ", descriptor.Description)
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

func (s *RelayService) update(descriptor service.ServiceDescriptor) {
	s.withWrite(func() {
		s.description = descriptor.Description
		s.status = descriptor.Status
		s.serviceType = descriptor.ServiceType
		s.executionType = descriptor.ExecutionType
		s.cTime = descriptor.CTime
		s.serviceUris = descriptor.ServiceUris
	})
}
