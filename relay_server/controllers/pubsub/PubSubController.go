package pubsub

import (
	"wsdk/common/logger"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/messages"
	"wsdk/relay_server/container"
	"wsdk/relay_server/context"
	"wsdk/relay_server/controllers/client_manager"
	"wsdk/relay_server/controllers/client_manager_v1"
	"wsdk/relay_server/controllers/connection_manager"
	"wsdk/relay_server/controllers/topic"
)

type PubSubController struct {
	topicManager  topic.ITopicManager                   `$inject:""`
	clientManager client_manager.IClientManager         `$inject:""`
	connManager   connection_manager.IConnectionManager `$inject:""`
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

func (c *PubSubController) getConnectionsAndSendByClientId(id string, message *messages.Message, onSendError func(connection.IConnection, error)) error {
	conns, err := c.connManager.GetConnectionsByClientId(id)
	if err != nil {
		return err
	}
	for _, conn := range conns {
		if cerr := conn.Send(message); cerr != nil {
			onSendError(conn, cerr)
		}
	}
	return nil
}

func (c *PubSubController) Publish(topicId string, message *messages.Message) error {
	topic, err := c.topicManager.GetTopic(topicId)
	if err != nil {
		return err
	}
	for _, subscriber := range topic.Subscribers() {
		if client := c.clientManager.GetClient(subscriber); client != nil {
			cerr := c.getConnectionsAndSendByClientId(client.Id(), message, func(conn connection.IConnection, err error) {
				c.logger.Printf("unable to broadcast message %d on topic %s to client %s(connection %s) due to %s", message.Id(), topicId, client.Id(), conn.Address(), err.Error())
			})
			if cerr != nil {
				c.logger.Printf("unable to broadcast message %d on topic %s to %s due to %s", message.Id(), topicId, client.Id(), cerr.Error())
				continue
			}
		}
	}
	return nil
}

func (c *PubSubController) Subscribe(clientId, topicId string) error {
	client := c.clientManager.GetClient(clientId)
	if client == nil {
		return client_manager_v1.NewClientNotFoundError(clientId)
	}
	return c.topicManager.SubscribeClientToTopic(client.Id(), topicId)
}

func (c *PubSubController) Unsubscribe(clientId, topicId string) error {
	client := c.clientManager.GetClient(clientId)
	if client == nil {
		return client_manager_v1.NewClientNotFoundError(clientId)
	}
	return c.topicManager.UnSubscribeClientToTopic(client.Id(), topicId)
}

func (c *PubSubController) Topics() ([]topic.TopicDescriptor, error) {
	return c.topicManager.GetAllDescribedTopics()
}

func (c *PubSubController) Remove(clientId, topicId string, removalMessage *messages.Message) error {
	client := c.clientManager.GetClient(clientId)
	if client == nil {
		return client_manager_v1.NewClientNotFoundError(clientId)
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
			err := s.getConnectionsAndSendByClientId(c.Id(),
				messages.DraftMessage(context.Ctx.Server().Id(), c.Id(), message.Uri(), message.MessageType(), message.Payload()),
				func(conn connection.IConnection, err error) {
					s.logger.Printf("err while sending topic %s removal message to client %s(connection %s) due to %s", topic.Id(), c.Id(), conn.Address(), err.Error())
				},
			)
			if err != nil {
				s.logger.Printf("err while sending topic %s removal message to client %s due to %s", topic.Id(), c.Id(), err.Error())
			}
		}
	}
}

func init() {
	container.Container.Singleton(func() IPubSubController {
		return NewPubSubController()
	})
}
