package pubsub

import (
	"encoding/json"
	"errors"
	"fmt"
	"wsdk/relay_common/messages"
	service_common "wsdk/relay_common/service"
	"wsdk/relay_server/container"
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
	RouteTopics = "/topics"
)

type PubSubService struct {
	*service.NativeService
	topics        map[string]*Topic
	clientManager managers.IClientManager
}

func New() *PubSubService {
	service := &PubSubService{
		NativeService: service.NewNativeService(ID, "message pub/sub service", service_common.ServiceTypeInternal, service_common.ServiceAccessTypeSocket, service_common.ServiceExecutionAsync),
		topics:        make(map[string]*Topic),
		clientManager: container.Container.GetById(managers.ClientManagerId).(managers.IClientManager),
	}
	service.initNotificationHandlers()
	service.initRoutes()
	return service
}

func (s *PubSubService) Init() error {
	s.NativeService = service.NewNativeService(ID, "message pub/sub service", service_common.ServiceTypeInternal, service_common.ServiceAccessTypeSocket, service_common.ServiceExecutionAsync)
	s.topics = make(map[string]*Topic)
	s.clientManager = container.Container.GetById(managers.ClientManagerId).(managers.IClientManager)
	if s.clientManager == nil {
		return errors.New("can not get clientManager from container")
	}
	s.initNotificationHandlers()
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

func (s *PubSubService) initNotificationHandlers() {
	events.OnEvent(events.EventClientDisconnected, func(e *messages.Message) {
		clientId := string(e.Payload()[:])
		for _, t := range s.topics {
			t.CheckAndRemoveSubscriber(clientId)
		}
	})
}

func (s *PubSubService) Subscribe(request *service_common.ServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	client := s.clientManager.GetClient(request.From())
	if client == nil {
		return errors.New(fmt.Sprintf("can not find client by id %s", request.From()))
	}
	topicId := pathParams[":topic"]
	topic := s.topics[topicId]
	if topic == nil {
		return errors.New(fmt.Sprintf("topic %s does not exist", topicId))
	}
	err := topic.CheckAndAddSubscriber(client.Id())
	if err != nil {
		return err
	}
	return s.ResolveByAck(request)
}

func (s *PubSubService) Unsubscribe(request *service_common.ServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	client := s.clientManager.GetClient(request.From())
	if client == nil {
		return errors.New(fmt.Sprintf("can not find client by id %s", request.From()))
	}
	topicId := pathParams[":topic"]
	topic := s.topics[topicId]
	if topic == nil {
		return errors.New(fmt.Sprintf("topic %s does not exist", topicId))
	}
	err := topic.CheckAndRemoveSubscriber(client.Id())
	if err != nil {
		return err
	}
	return s.ResolveByAck(request)
}

func (s *PubSubService) Publish(request *service_common.ServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	client := s.clientManager.GetClient(request.From())
	if client == nil {
		return errors.New(fmt.Sprintf("can not find client by id %s", request.From()))
	}
	topicId := pathParams[":topic"]
	topic := s.topics[topicId]
	if topic == nil {
		return errors.New(fmt.Sprintf("topic %s does not exist", topicId))
	}
	subscribers := topic.Subscribers()
	for _, subscriber := range subscribers {
		if c := s.clientManager.GetClient(subscriber); c != nil {
			c.Send(request.Message)
		}
	}
	return s.ResolveByAck(request)
}

func (s *PubSubService) Topics(request *service_common.ServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	if !s.clientManager.HasClient(request.From()) {
		return errors.New(fmt.Sprintf("can not find client by id %s", request.From()))
	}
	topics := make([]TopicDescriptor, 0, len(s.topics))
	for _, t := range s.topics {
		topics = append(topics, t.Describe())
	}
	marshalled, err := json.Marshal(topics)
	if err != nil {
		return err
	}
	return request.Resolve(messages.NewMessage(request.Id(), s.HostInfo().Id, request.From(), request.Uri(), messages.MessageTypeServiceResponse, marshalled))
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
	topic := s.topics[topicId]
	if topic == nil {
		return errors.New(fmt.Sprintf("topic %s does not exist", topicId))
	}
	if err := topic.CheckAndRemoveSubscriber(client.Id()); err != nil {
		return err
	}
	s.notifySubscribersForTopicRemoval(topic, request.Message)
	return nil
}
