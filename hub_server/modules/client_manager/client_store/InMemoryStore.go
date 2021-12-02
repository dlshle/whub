package client_store

import (
	"errors"
	"sync"
	"whub/hub_server/client"
)

type InMemoryStore struct {
	lock    *sync.RWMutex
	clients map[string]*client.Client
}

func (s *InMemoryStore) withWrite(cb func()) {
	s.lock.Lock()
	defer s.lock.Unlock()
	cb()
}

func (s *InMemoryStore) withRead(cb func()) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	cb()
}

func (s *InMemoryStore) Get(id string) (c *client.Client, e error) {
	s.withRead(func() {
		c = s.clients[id]
		if c == nil {
			e = errors.New("not found")
		}
	})
	return
}

func (s *InMemoryStore) GetAll() (clients []*client.Client, e error) {
	s.withRead(func() {
		var allClients []*client.Client
		for _, v := range s.clients {
			allClients = append(allClients, v)
		}
		clients = allClients
	})
	return
}

func (s *InMemoryStore) Create(client *client.Client) (err error) {
	s.withWrite(func() {
		if s.clients[client.Id()] != nil {
			err = errors.New("client already exist")
			return
		}
		s.clients[client.Id()] = client
	})
	return
}

func (s *InMemoryStore) Update(client *client.Client) (err error) {
	s.withWrite(func() {
		if s.clients[client.Id()] == nil {
			err = errors.New("not found")
			return
		}
		s.clients[client.Id()] = client
	})
	return
}

func (s *InMemoryStore) Has(id string) (res bool, err error) {
	s.withRead(func() {
		res = s.clients[id] != nil
	})
	return
}

func (s *InMemoryStore) Delete(id string) (err error) {
	s.withWrite(func() {
		if s.clients[id] == nil {
			err = errors.New("not found")
			return
		}
		delete(s.clients, id)
	})
	return
}

func (s *InMemoryStore) Find(query *DClientQuery) ([]*client.Client, error) {
	panic("implement me")
}

func (s *InMemoryStore) Close() error {
	return nil
}

func NewInMemoryStore() IClientStore {
	return &InMemoryStore{
		lock:    new(sync.RWMutex),
		clients: make(map[string]*client.Client),
	}
}
