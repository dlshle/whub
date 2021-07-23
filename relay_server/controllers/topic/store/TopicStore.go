package store

import (
	"wsdk/relay_server/controllers/topic"
	error2 "wsdk/relay_server/controllers/topic/error"
)

type ITopicStore interface {
	Has(id string) (bool, error2.ITopicError)
	Create(id string, creatorClientId string) (*topic.Topic, error2.ITopicError)
	Update(topic *topic.Topic) error2.ITopicError
	Get(id string) (*topic.Topic, error2.ITopicError)
	Delete(id string) error2.ITopicError
	Topics() ([]*topic.Topic, error2.ITopicError)
}
