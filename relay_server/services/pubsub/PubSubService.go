package pubsub

import (
	"encoding/json"
	"errors"
	service_common "wsdk/relay_common/service"
	"wsdk/relay_server/container"
	"wsdk/relay_server/controllers/pubsub"
	"wsdk/relay_server/service_base"
)

const (
	ID          = "pubsub"
	RouteSub    = "/subscribe/:topic"
	RouteUnSub  = "/unsubscribe/:topic"
	RoutePub    = "/publish/:topic"
	RouteRemove = "/remove/:topic"
	RouteTopics = "/topics"
)

// TODO use PubSubController instead doing business here
type PubSubService struct {
	*service_base.NativeService
	pubSubController pubsub.IPubSubController `$inject:""`
}

func (s *PubSubService) Init() error {
	s.NativeService = service_base.NewNativeService(ID, "message pub/sub service", service_common.ServiceTypeInternal, service_common.ServiceAccessTypeSocket, service_common.ServiceExecutionAsync)
	err := container.Container.Fill(s)
	if err != nil {
		return err
	}
	if s.pubSubController == nil {
		return errors.New("can not get PubSubController from container")
	}
	return s.initRoutes()
}

func (s *PubSubService) initRoutes() (err error) {
	err = s.RegisterRoute(RouteSub, s.Subscribe)
	if err != nil {
		return
	}
	err = s.RegisterRoute(RouteUnSub, s.Unsubscribe)
	if err != nil {
		return
	}
	err = s.RegisterRoute(RoutePub, s.Publish)
	if err != nil {
		return
	}
	err = s.RegisterRoute(RouteRemove, s.Remove)
	if err != nil {
		return
	}
	err = s.RegisterRoute(RouteTopics, s.Topics)
	return
}

func (s *PubSubService) Subscribe(request *service_common.ServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	err := s.pubSubController.Subscribe(request.From(), pathParams["topic"])
	if err != nil {
		return err
	}
	s.ResolveByAck(request)
	return nil
}

func (s *PubSubService) Unsubscribe(request *service_common.ServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	err := s.pubSubController.Unsubscribe(request.From(), pathParams["topic"])
	if err != nil {
		return err
	}
	s.ResolveByAck(request)
	return nil
}

func (s *PubSubService) Publish(request *service_common.ServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	err := s.pubSubController.Publish(pathParams["topic"], request.Message)
	if err != nil {
		return err
	}
	s.ResolveByAck(request)
	return nil
}

func (s *PubSubService) Topics(request *service_common.ServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	topics, err := s.pubSubController.Topics()
	if err != nil {
		return err
	}
	marshalled, err := json.Marshal(topics)
	if err != nil {
		return err
	}
	s.ResolveByResponse(request, marshalled)
	return nil
}

func (s *PubSubService) Remove(request *service_common.ServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	err := s.pubSubController.Remove(request.From(), pathParams["topic"], request.Message)
	if err != nil {
		return err
	}
	s.ResolveByAck(request)
	return nil
}
