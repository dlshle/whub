package pubsub

import (
	"encoding/json"
	"errors"
	"fmt"
	"wsdk/relay_common/messages"
	service_common "wsdk/relay_common/service"
	"wsdk/relay_server/container"
	"wsdk/relay_server/context"
	"wsdk/relay_server/controllers/client_manager"
	"wsdk/relay_server/controllers/topic"
	topic_error "wsdk/relay_server/controllers/topic/error"
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
	topicManager  topic.ITopicManager           `$inject:""`
	clientManager client_manager.IClientManager `$inject:""`
}

func (s *PubSubService) Init() error {
	s.NativeService = service_base.NewNativeService(ID, "message pub/sub service_manager", service_common.ServiceTypeInternal, service_common.ServiceAccessTypeSocket, service_common.ServiceExecutionAsync)
	container.Container.Fill(s)
	if s.clientManager == nil || s.topicManager == nil {
		return errors.New("can not get clientManager from container")
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
	client := s.clientManager.GetClient(request.From())
	if client == nil {
		return errors.New(fmt.Sprintf("can not find client_manager by id %s", request.From()))
	}
	topicId := pathParams[":topic"]
	err := s.topicManager.SubscribeClientToTopic(client.Id(), topicId)
	if err != nil {
		return err
	}
	s.ResolveByAck(request)
	return nil
}

func (s *PubSubService) Unsubscribe(request *service_common.ServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	client := s.clientManager.GetClient(request.From())
	if client == nil {
		return errors.New(fmt.Sprintf("can not find client_manager by id %s", request.From()))
	}
	topicId := pathParams[":topic"]
	err := s.topicManager.UnSubscribeClientToTopic(client.Id(), topicId)
	if err != nil {
		return err
	}
	s.ResolveByAck(request)
	return nil
}

func (s *PubSubService) Publish(request *service_common.ServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	client := s.clientManager.GetClient(request.From())
	if client == nil {
		return errors.New(fmt.Sprintf("can not find client_manager by id %s", request.From()))
	}
	topicId := pathParams[":topic"]
	t, err := s.topicManager.GetTopic(topicId)
	if err.Code() == topic_error.TopicErrNotFound {
		// create topic
		t, err = s.topicManager.CreateTopic(topicId, client.Id())
		if err != nil {
			return err
		}
	}
	subscribers := t.Subscribers()
	for _, subscriber := range subscribers {
		if c := s.clientManager.GetClient(subscriber); c != nil {
			c.Send(request.Message)
		}
	}
	s.ResolveByAck(request)
	return nil
}

func (s *PubSubService) Topics(request *service_common.ServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	if !s.clientManager.HasClient(request.From()) {
		return errors.New(fmt.Sprintf("can not find client_manager by id %s", request.From()))
	}
	topics, terr := s.topicManager.GetAllDescribedTopics()
	if terr != nil {
		return terr
	}
	marshalled, err := json.Marshal(topics)
	if err != nil {
		return err
	}
	request.Resolve(messages.NewMessage(request.Id(), s.HostInfo().Id, request.From(), request.Uri(), messages.MessageTypeServiceResponse, marshalled))
	return nil
}

func (s *PubSubService) notifySubscribersForTopicRemoval(topic topic.Topic, message *messages.Message) {
	subscribers := topic.Subscribers()
	for _, subscriber := range subscribers {
		if c := s.clientManager.GetClient(subscriber); c != nil {
			err := c.Send(messages.DraftMessage(context.Ctx.Server().Id(), c.Id(), message.Uri(), message.MessageType(), message.Payload()))
			if err != nil {
				s.Logger().Printf("err while sending topic %s removal message to %s", topic.Id(), c.Id())
			}
		}
	}
}

func (s *PubSubService) Remove(request *service_common.ServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	client := s.clientManager.GetClient(request.From())
	if client == nil {
		return errors.New(fmt.Sprintf("can not find client_manager by id %s", request.From()))
	}
	topicId := pathParams[":topic"]
	topic, err := s.topicManager.GetTopic(topicId)
	if err != nil {
		return err
	}
	err = s.topicManager.RemoveTopic(topicId, client.Id())
	if err != nil {
		return err
	}
	s.notifySubscribersForTopicRemoval(topic, request.Message)
	s.ResolveByAck(request)
	return nil
}
