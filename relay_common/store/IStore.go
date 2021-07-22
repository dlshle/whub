package store

type IStore interface {
	Put(id string, item interface{}) error
	Get(id string) (interface{}, error)
	Delete(id string) error
}
