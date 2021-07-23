package error

import (
	"fmt"
	"wsdk/relay_server/controllers/topic"
)

const (
	TopicErrInsufficientPermission = 3
	TopicErrNotValidSubscriber     = 2
	TopicErrAlreadySubscribed      = 1
	TopicErrExceededMaxSubscriber  = 0
	TopicErrNotFound               = -1
	TopicErrExceededCacheSize      = -2
)

type TopicError struct {
	msg  string
	code int
}

type ITopicError interface {
	Error() string
	Code() int
}

func (e *TopicError) Error() string {
	return e.msg
}

func (e *TopicError) Code() int {
	return e.code
}

func NewTopicError(code int, msg string) ITopicError {
	return &TopicError{
		code: code,
		msg:  msg,
	}
}

func NewTopicNotFoundError(id string) ITopicError {
	return NewTopicError(TopicErrNotFound, fmt.Sprintf("topic %s is not found", id))
}

func NewTopicCacheSizeExceededError(cacheSize int) ITopicError {
	return NewTopicError(TopicErrExceededCacheSize, fmt.Sprintf("topic store size exceeded maxCacheSize %d", cacheSize))
}

func NewTopicMaxSubscribersExceededError(id string) ITopicError {
	return NewTopicError(TopicErrExceededMaxSubscriber, fmt.Sprintf("number of subscribers exceeded max subscribers count %d for topic %s", topic.MaxSubscribersPerTopic, id))
}

func NewTopicClientNotValidSubscriberError(topicId string, clientId string) ITopicError {
	return NewTopicError(TopicErrNotValidSubscriber, fmt.Sprintf("client_manager %s is not a subscriber of topic %s", clientId, topicId))
}
func NewTopicClientAlreadySubscribedError(topicId string, clientId string) ITopicError {
	return NewTopicError(TopicErrAlreadySubscribed, fmt.Sprintf("subscriber %s has already subscriberd to topic %s", clientId, topicId))
}

func NewTopicClientInsufficientPermissionError(topicId string, clientId string, permission string) ITopicError {
	return NewTopicError(TopicErrInsufficientPermission, fmt.Sprintf("client_manager %s does not have [%s] permission for topic %s", clientId, permission, topicId))
}
