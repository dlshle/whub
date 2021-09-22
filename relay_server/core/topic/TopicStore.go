package topic

import (
	error2 "wsdk/relay_server/core/error"
)

type ITopicStore interface {
	Has(id string) (bool, error2.IControllerError)
	Create(id string, creatorClientId string) (*Topic, error2.IControllerError)
	Update(topic *Topic) error2.IControllerError
	Get(id string) (*Topic, error2.IControllerError)
	Delete(id string) error2.IControllerError
	Topics() ([]*Topic, error2.IControllerError)
}
