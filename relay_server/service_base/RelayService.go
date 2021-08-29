package service_base

import (
	"errors"
	"fmt"
	"wsdk/common/utils"
	"wsdk/relay_common/service"
	"wsdk/relay_server/client"
	"wsdk/relay_server/request"
)

type RelayService struct {
	*Service
	executor *request.RelayServiceRequestExecutor
}

type IRelayService interface {
	IService
	Init(descriptor service.ServiceDescriptor,
		provider IServiceProvider,
		executor *request.RelayServiceRequestExecutor)
	RestoreExternally(reconnectedOwner *client.Client) error
	Update(descriptor service.ServiceDescriptor) error
	UpdateProviderConnection(connAddr string) error
}

func (s *RelayService) Init(
	descriptor service.ServiceDescriptor,
	provider IServiceProvider,
	executor *request.RelayServiceRequestExecutor) {
	s.Service = NewService(descriptor.Id, descriptor.Description, provider, executor, descriptor.ServiceUris, descriptor.ServiceType, descriptor.AccessType, descriptor.ExecutionType)
	s.executor = executor
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
		s.serviceQueue = service.NewServiceTaskQueue(s.HostInfo().Id, request.NewRelayServiceRequestExecutor(s.Id(), s.Provider().Id()), s.ctx.ServiceTaskPool())
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

func (s *RelayService) UpdateProviderConnection(connAddr string) error {
	if s.Status() >= service.ServiceStatusStopping {
		return errors.New(fmt.Sprintf("invalid service status for update provider connection(%d)", s.Status()))
	}
	return s.executor.UpdateProviderConnection(connAddr)
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
