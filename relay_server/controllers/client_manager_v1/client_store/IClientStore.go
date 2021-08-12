package client_store

import "wsdk/relay_server/client"

const (
	QueryCondId = "Id"
)

type IQuery interface {
	OnCond(condition string, value interface{}) IQuery
}

type IClientStore interface {
	Get(id string) (*client.Client, error)
	Put(id string, client *client.Client) error
	Has(id string) (bool, error)
	Delete(id string) error
	Find(query IQuery) (*client.Client, error)
}
