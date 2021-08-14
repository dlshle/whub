package topic

import (
	"wsdk/relay_server/core"
)

type ITopicStore interface {
	Has(id string) (bool, core.IControllerError)
	Create(id string, creatorClientId string) (*Topic, core.IControllerError)
	Update(topic *Topic) core.IControllerError
	Get(id string) (*Topic, core.IControllerError)
	Delete(id string) core.IControllerError
	Topics() ([]*Topic, core.IControllerError)
}
