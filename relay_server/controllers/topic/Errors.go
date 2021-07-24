package topic

import (
	"fmt"
	"wsdk/relay_server/controllers"
)

const (
	TopicErrInsufficientPermission = 501
	TopicErrNotValidSubscriber     = 502
	TopicErrAlreadySubscribed      = 503
	TopicErrExceededMaxSubscriber  = 504
	TopicErrNotFound               = 505
	TopicErrExceededCacheSize      = 506
)

func NewTopicNotFoundError(id string) controllers.IControllerError {
	return controllers.NewControllerError(TopicErrNotFound, fmt.Sprintf("topic %s is not found", id))
}

func NewTopicCacheSizeExceededError(cacheSize int) controllers.IControllerError {
	return controllers.NewControllerError(TopicErrExceededCacheSize, fmt.Sprintf("topic store size exceeded maxCacheSize %d", cacheSize))
}

func NewTopicMaxSubscribersExceededError(id string, maxSubscribersPerTopic int) controllers.IControllerError {
	return controllers.NewControllerError(TopicErrExceededMaxSubscriber, fmt.Sprintf("number of subscribers exceeded max subscribers count %d for topic %s", maxSubscribersPerTopic, id))
}

func NewTopicClientNotValidSubscriberError(topicId string, clientId string) controllers.IControllerError {
	return controllers.NewControllerError(TopicErrNotValidSubscriber, fmt.Sprintf("client_manager %s is not a subscriber of topic %s", clientId, topicId))
}
func NewTopicClientAlreadySubscribedError(topicId string, clientId string) controllers.IControllerError {
	return controllers.NewControllerError(TopicErrAlreadySubscribed, fmt.Sprintf("subscriber %s has already subscriberd to topic %s", clientId, topicId))
}

func NewTopicClientInsufficientPermissionError(topicId string, clientId string, permission string) controllers.IControllerError {
	return controllers.NewControllerError(TopicErrInsufficientPermission, fmt.Sprintf("client_manager %s does not have [%s] permission for topic %s", clientId, permission, topicId))
}
