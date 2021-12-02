package client_manager

import (
	"errors"
	"time"
	"whub/common/logger"
	"whub/hub_server/client"
	"whub/hub_server/config"
	"whub/hub_server/module_base"
	"whub/hub_server/modules/client_manager/client_store"
)

const ID = "ClientManager"

type IClientManagerModule interface {
	HasClient(id string) (bool, error)
	GetClient(id string) (*client.Client, error)
	GetClientWithErrOnNotFound(id string) (c *client.Client, e error)
	WithAllClients(cb func(clients []*client.Client)) error
	AddClient(client *client.Client) error
	UpdateClient(client *client.Client) error
	DeleteClient(id string) error
	GetClientsByType(cType int) ([]*client.Client, error)
	GetClientsCreatedAfter(time time.Time) ([]*client.Client, error)
	GetClientsCreatedBefore(time time.Time) ([]*client.Client, error)
	GetAllClients() ([]*client.Client, error)
}

type ClientManagerModule struct {
	*module_base.ModuleBase
	store  client_store.IClientStore // need to use IOC later, use SQLStore now for test
	logger *logger.SimpleLogger
}

func (c *ClientManagerModule) Init() error {
	c.ModuleBase = module_base.NewModuleBase(ID, func() error {
		return c.store.Close()
	})
	c.logger = c.Logger()
	c.store = createClientManagerStore(c.logger)
	return nil
}

func createCachedStore(clientManagerConfig config.DomainConfig) (client_store.IClientStore, error) {
	mySqlConfig := clientManagerConfig.Persistent
	redisConfig := clientManagerConfig.Redis
	if mySqlConfig.Driver != "mysql" {
		return nil, errors.New("invalid clientManager.persistent.db value")
	}
	store := client_store.NewCachedClientMySqlStore()
	err := store.Init(mySqlConfig.Server, mySqlConfig.Username, mySqlConfig.Password, mySqlConfig.Db, redisConfig.Server, redisConfig.Password)
	return store, err
}

func createMySqlStore(clientManagerConfig config.DomainConfig) (client_store.IClientStore, error) {
	mySqlConfig := clientManagerConfig.Persistent
	if mySqlConfig.Driver != "mysql" {
		return nil, errors.New("invalid clientManager.persistent.db value")
	}
	store := client_store.NewMySqlClientStore()
	err := store.Init(mySqlConfig.Server, mySqlConfig.Username, mySqlConfig.Password, mySqlConfig.Db)
	return store, err
}

func createInMemoryStore() (client_store.IClientStore, error) {
	return client_store.NewInMemoryStore(), nil
}

func createClientManagerStore(logger *logger.SimpleLogger) client_store.IClientStore {
	domainConfig := config.Config.DomainConfigs
	clientManagerConfig := domainConfig["clientManager"]
	hasPersistent := clientManagerConfig.Persistent.Server != ""
	hasRedis := clientManagerConfig.Redis.Server != ""
	var store client_store.IClientStore
	var err error
	if hasPersistent && hasRedis {
		logger.Printf("create cached mysql client store with redisServer %s and mySqlServer %s", clientManagerConfig.Redis.Server, clientManagerConfig.Persistent.Server)
		store, err = createCachedStore(clientManagerConfig)
	} else if hasPersistent {
		logger.Printf("create mysql client store with  mySqlServer %s", clientManagerConfig.Persistent.Server)
		store, err = createMySqlStore(clientManagerConfig)
	} else {
		logger.Println("create in memory client store")
		store, err = createInMemoryStore()
	}
	if err != nil {
		logger.Printf("create configured store failed due to %s, will use in memory store for ClientManagerModule", err.Error())
		store, err = createInMemoryStore()
	}
	return store
}

func (m *ClientManagerModule) HasClient(id string) (bool, error) {
	return m.store.Has(id)
}

func (m *ClientManagerModule) GetClient(id string) (c *client.Client, e error) {
	if id == "" {
		return nil, NewInvalidClientIdError("")
	}
	return m.store.Get(id)
}

func (m *ClientManagerModule) GetClientWithErrOnNotFound(id string) (c *client.Client, e error) {
	c, e = m.GetClient(id)
	if e == nil && c == nil {
		e = NewClientNotFoundError(id)
	}
	return
}

func (m *ClientManagerModule) WithAllClients(cb func(clients []*client.Client)) error {
	allClients, err := m.store.GetAll()
	if err != nil {
		return err
	}
	cb(allClients)
	return nil
}

func (m *ClientManagerModule) AddClient(client *client.Client) error {
	return m.store.Create(client)
}

func (m *ClientManagerModule) UpdateClient(client *client.Client) error {
	return m.store.Update(client)
}

func (m *ClientManagerModule) DeleteClient(id string) error {
	return m.store.Delete(id)
}

func (m *ClientManagerModule) GetClientsByType(cType int) ([]*client.Client, error) {
	return m.store.Find(client_store.Query().Type(cType))
}

func (m *ClientManagerModule) GetClientsCreatedAfter(time time.Time) ([]*client.Client, error) {
	return m.store.Find(client_store.Query().CreatedAfter(time))
}

func (m *ClientManagerModule) GetClientsCreatedBefore(time time.Time) ([]*client.Client, error) {
	return m.store.Find(client_store.Query().CreatedBefore(time))
}

func (m *ClientManagerModule) GetAllClients() ([]*client.Client, error) {
	return m.store.GetAll()
}
