package topic

import (
	"wsdk/relay_server/controllers"
)

type ITopicStore interface {
	Has(id string) (bool, controllers.IControllerError)
	Create(id string, creatorClientId string) (*Topic, controllers.IControllerError)
	Update(topic *Topic) controllers.IControllerError
	Get(id string) (*Topic, controllers.IControllerError)
	Delete(id string) controllers.IControllerError
	Topics() ([]*Topic, controllers.IControllerError)
}
