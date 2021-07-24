package pubsub

import (
	"wsdk/common/logger"
	"wsdk/relay_common/messages"
	"wsdk/relay_server/container"
	"wsdk/relay_server/context"
	"wsdk/relay_server/controllers/client_manager"
	"wsdk/relay_server/controllers/topic"
)

type PubSubController struct {
	topicManager  topic.ITopicManager           `$inject:""`
	clientManager client_manager.IClientManager `$inject:""`
	logger        *logger.SimpleLogger
}

type IPubSubController interface {
	Publish(topicId string, message *messages.Message) error
	Subscribe(clientId, topicId string) error
	Unsubscribe(clientId, topicId string) error
	Topics() ([]topic.TopicDescriptor, error)
	Remove(clientId, topicId string, removalMessage *messages.Message) error
}

func NewPubSubController() IPubSubController {
	c := &PubSubController{
		logger: context.Ctx.Logger().WithPrefix("[PubSubController]"),
	}
	err := container.Container.Fill(c)
	if err != nil {
		panic(err)
	}
	return c
}

func (c *PubSubController) Publish(topicId string, message *messages.Message) error {
	topic, err := c.topicManager.GetTopic(topicId)
	if err != nil {
		return err
	}
	for _, subscriber := range topic.Subscribers() {
		if client := c.clientManager.GetClient(subscriber); client != nil {
			if cerr := client.Send(message); cerr != nil {
				c.logger.Printf("unable to broadcast message %d on topic %s to %s due to %s", message.Id(), topicId, client.Id(), cerr.Error())
			}
		}
	}
	return nil
}

func (c *PubSubController) Subscribe(clientId, topicId string) error {
	client := c.clientManager.GetClient(clientId)
	if client == nil {
		return client_manager.NewClientNotFoundError(clientId)
	}
	return c.topicManager.SubscribeClientToTopic(client.Id(), topicId)
}

func (c *PubSubController) Unsubscribe(clientId, topicId string) error {
	client := c.clientManager.GetClient(clientId)
	if client == nil {
		return client_manager.NewClientNotFoundError(clientId)
	}
	return c.topicManager.UnSubscribeClientToTopic(client.Id(), topicId)
}

func (c *PubSubController) Topics() ([]topic.TopicDescriptor, error) {
	return c.topicManager.GetAllDescribedTopics()
}

func (c *PubSubController) Remove(clientId, topicId string, removalMessage *messages.Message) error {
	client := c.clientManager.GetClient(clientId)
	if client == nil {
		return client_manager.NewClientNotFoundError(clientId)
	}
	topic, err := c.topicManager.GetTopic(topicId)
	if err != nil {
		return err
	}
	err = c.topicManager.RemoveTopic(topicId, clientId)
	if err != nil {
		return err
	}
	c.notifySubscribersForTopicRemoval(topic, removalMessage)
	return nil
}

func (s *PubSubController) notifySubscribersForTopicRemoval(topic topic.Topic, message *messages.Message) {
	subscribers := topic.Subscribers()
	for _, subscriber := range subscribers {
		if c := s.clientManager.GetClient(subscriber); c != nil {
			err := c.Send(messages.DraftMessage(context.Ctx.Server().Id(), c.Id(), message.Uri(), message.MessageType(), message.Payload()))
			if err != nil {
				s.logger.Printf("err while sending topic %s removal message to %s", topic.Id(), c.Id())
			}
		}
	}
}

func init() {
	container.Container.Singleton(func() IPubSubController {
		return NewPubSubController()
	})
}
