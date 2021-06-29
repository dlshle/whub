package pubsub

import (
	"errors"
	"fmt"
	"wsdk/relay_common/messages"
	service_common "wsdk/relay_common/service"
	"wsdk/relay_server/context"
	"wsdk/relay_server/events"
	"wsdk/relay_server/managers"
	"wsdk/relay_server/service"
)

const (
	ID          = "pubsub"
	RouteSub    = "/subscribe/:topic"
	RouteUnSub  = "/unsubscribe/:topic"
	RoutePub    = "/publish/:topic"
	RouteRemove = "/remove/:topic"
)

type PubSubService struct {
	*service.NativeService
	topicSubscribers map[string]*Topic
	clientManager    managers.IClientManager `autowire`
}

func New() *PubSubService {
	// TODO New func
	return &PubSubService{}
}

func (s *PubSubService) registerRoutes() {
	s.RegisterRoute(RouteSub, s.Subscribe)
	s.RegisterRoute(RouteUnSub, s.Unsubscribe)
	s.RegisterRoute(RoutePub, s.Publish)
	s.RegisterRoute(RouteRemove, s.Remove)
}

func (s *PubSubService) initNotifications() {
	context.Ctx.NotificationEmitter().On(events.EventClientDisconnected, func(e *messages.Message) {
		clientId := string(e.Payload()[:])
		for _, t := range s.topicSubscribers {
			t.removeSubscriber(clientId)
		}
	})
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

func (s *PubSubService) Publish(request *service_common.ServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	client := s.clientManager.GetClient(request.From())
	if client == nil {
		return errors.New(fmt.Sprintf("can not find client by id %s", request.From()))
	}
	topicId := pathParams[":topic"]
	topic := s.topicSubscribers[topicId]
	if topic == nil {
		return errors.New(fmt.Sprintf("topic %s does not exist", topicId))
	}
	subscribers := topic.Subscribers()
	for _, subscriber := range subscribers {
		if c := s.clientManager.GetClient(subscriber); c != nil {
			c.Send(request.Message)
		}
	}
	return nil
}

func (s *PubSubService) notifySubscribersForTopicRemoval(topic *Topic, message *messages.Message) {
	subscribers := topic.Subscribers()
	for _, subscriber := range subscribers {
		if c := s.clientManager.GetClient(subscriber); c != nil {
			c.Send(messages.DraftMessage(context.Ctx.Server().Id(), c.Id(), message.Uri(), message.MessageType(), message.Payload()))
		}
	}
}

func (s *PubSubService) Remove(request *service_common.ServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	client := s.clientManager.GetClient(request.From())
	if client == nil {
		return errors.New(fmt.Sprintf("can not find client by id %s", request.From()))
	}
	topicId := pathParams[":topic"]
	topic := s.topicSubscribers[topicId]
	if topic == nil {
		return errors.New(fmt.Sprintf("topic %s does not exist", topicId))
	}
	if err := topic.CheckAndRemoveSubscriber(client.Id()); err != nil {
		return err
	}
	s.notifySubscribersForTopicRemoval(topic, request.Message)
	return nil
}
