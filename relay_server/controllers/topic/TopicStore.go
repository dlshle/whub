package topic

type ITopicStore interface {
	Has(id string) (bool, error)
	Create(id string, creatorClientId string) (Topic, error)
	Update(topic Topic) error
	Get(id string) (Topic, error)
	Delete(id string) (Topic, error)
}
