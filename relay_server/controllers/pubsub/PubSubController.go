package pubsub

import (
	"wsdk/common/logger"
	"wsdk/relay_common/messages"
	"wsdk/relay_server/controllers/client_manager"
	"wsdk/relay_server/controllers/topic"
)

type PubSubController struct {
	topicManager  topic.ITopicManager           `$inject:""`
	clientManager client_manager.IClientManager `$inject:""`
	logger        *logger.SimpleLogger
}

type IPubSubController interface {
	Publish(topicId string, from string, message []byte) error
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
