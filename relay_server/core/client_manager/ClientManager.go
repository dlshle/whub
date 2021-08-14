package client_manager

import (
	"time"
	"wsdk/common/logger"
	"wsdk/relay_server/client"
	"wsdk/relay_server/container"
	"wsdk/relay_server/context"
	"wsdk/relay_server/core/client_manager/client_store"
)

type IClientManager interface {
	HasClient(id string) (bool, error)
	GetClient(id string) (*client.Client, error)
	WithAllClients(cb func(clients []*client.Client)) error
	AddClient(client *client.Client) error
	UpdateClient(client *client.Client) error
	DeleteClient(id string) error
	GetClientsByType(cType int) ([]*client.Client, error)
	GetClientsCreatedAfter(time time.Time) ([]*client.Client, error)
	GetClientsCreatedBefore(time time.Time) ([]*client.Client, error)
}

type ClientManager struct {
	store  client_store.IClientStore // need to use IOC later, use SQLStore now for test
	logger *logger.SimpleLogger
}

func NewClientManager() IClientManager {
	// TODO remove later test only
	sqlStore := client_store.NewMySqlClientStore()
	err := sqlStore.Init(client_store.SQLServer, client_store.SQLUserName, client_store.SQLPassword, client_store.SQLDBName)
	if err != nil {
		panic(err)
	}
	// ^^ TEST ONLY
	manager := &ClientManager{
		store:  sqlStore,
		logger: context.Ctx.Logger().WithPrefix("[ClientManager]"),
	}
	// err := container.Container.Fill(manager)
	// if err != nil {
	// panic(err)
	// }
	return manager
}

func (m *ClientManager) HasClient(id string) (bool, error) {
	return m.store.Has(id)
}

func (m *ClientManager) GetClient(id string) (c *client.Client, e error) {
	c, e = m.store.Get(id)
	if c == nil {
		e = NewClientNotFoundError(id)
	}
	return
}

func (m *ClientManager) WithAllClients(cb func(clients []*client.Client)) error {
	allClients, err := m.store.GetAll()
	if err != nil {
		return err
	}
	cb(allClients)
	return nil
}

func (m *ClientManager) AddClient(client *client.Client) error {
	return m.store.Create(client)
}

func (m *ClientManager) UpdateClient(client *client.Client) error {
	return m.store.Update(client)
}

func (m *ClientManager) DeleteClient(id string) error {
	return m.store.Delete(id)
}

func (m *ClientManager) GetClientsByType(cType int) ([]*client.Client, error) {
	return m.store.Find(client_store.Query().Type(cType))
}

func (m *ClientManager) GetClientsCreatedAfter(time time.Time) ([]*client.Client, error) {
	return m.store.Find(client_store.Query().CreatedAfter(time))
}

func (m *ClientManager) GetClientsCreatedBefore(time time.Time) ([]*client.Client, error) {
	return m.store.Find(client_store.Query().CreatedBefore(time))
}

func init() {
	container.Container.Singleton(func() IClientManager {
		return NewClientManager()
	})
}
