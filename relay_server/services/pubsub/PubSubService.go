package pubsub

import (
	"errors"
	"fmt"
	service_common "wsdk/relay_common/service"
	"wsdk/relay_server/managers"
)

const (
	ID         = "pubsub"
	RouteSub   = "/subscribe/:topic"
	RouteUnSub = "/unsubscribe/:topic"
	RoutePub   = "/publish/:topic"
)

type PubSubService struct {
	topicSubscribers map[string]*Topic
	clientManager    managers.IClientManager `autowire`
}

func New() *PubSubService {
	return &PubSubService{}
}

func (s *PubSubService) initNotifications() {

}

func (s *PubSubService) Subscribe(request *service_common.ServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	client := s.clientManager.GetClient(request.From())
	if client == nil {
		return errors.New(fmt.Sprintf("can not find client by id %s", request.From()))
	}
	topicId := pathParams[":topic"]
	topic := s.topicSubscribers[topicId]
	if topic == nil {
		return errors.New(fmt.Sprintf("topic %s does not exist", topicId))
	}
	return topic.CheckAndAddSubscriber(client.Id())
}

func (s *PubSubService) Unsubscribe(request *service_common.ServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	client := s.clientManager.GetClient(request.From())
	if client == nil {
		return errors.New(fmt.Sprintf("can not find client by id %s", request.From()))
	}
	topicId := pathParams[":topic"]
	topic := s.topicSubscribers[topicId]
	if topic == nil {
		return errors.New(fmt.Sprintf("topic %s does not exist", topicId))
	}
	return topic.CheckAndRemoveSubscriber(client.Id())
}
