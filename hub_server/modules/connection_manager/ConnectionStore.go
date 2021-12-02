package connection_manager

import (
	"errors"
	"fmt"
	"sync"
	"whub/hub_common/connection"
)

type IConnectionStore interface {
	Add(connection.IConnection) error
	Has(string) (bool, error)
	Delete(string) error
	Get(string) (connection.IConnection, error)
	GetAll() ([]connection.IConnection, error)
}

type InMemoryConnectionStore struct {
	store  map[string]connection.IConnection
	rwLock *sync.RWMutex
}

func NewInMemoryConnectionStore() IConnectionStore {
	return &InMemoryConnectionStore{
		store:  make(map[string]connection.IConnection),
		rwLock: new(sync.RWMutex),
	}
}

func (s *InMemoryConnectionStore) withWrite(cb func()) {
	s.rwLock.Lock()
	defer s.rwLock.Unlock()
	cb()
}

func (s *InMemoryConnectionStore) withRead(cb func()) {
	s.rwLock.RLock()
	defer s.rwLock.RUnlock()
	cb()
}

func (s *InMemoryConnectionStore) Add(c connection.IConnection) error {
	if c == nil {
		return errors.New("nil connection")
	}
	if exist, _ := s.Has(c.Address()); exist {
		return errors.New(fmt.Sprintf("connection %s already exists", c.Address()))
	}
	s.withWrite(func() {
		s.store[c.Address()] = c
	})
	return nil
}

func (s *InMemoryConnectionStore) Has(addr string) (bool, error) {
	c, err := s.Get(addr)
	return c != nil, err
}

func (s *InMemoryConnectionStore) Delete(addr string) error {
	if exist, _ := s.Has(addr); !exist {
		return errors.New(fmt.Sprintf("connection from %s does not exist", addr))
	}
	s.withWrite(func() {
		delete(s.store, addr)
	})
	return nil
}

func (s *InMemoryConnectionStore) Get(addr string) (conn connection.IConnection, err error) {
	s.withRead(func() {
		conn = s.store[addr]
	})
	return
}

func (s *InMemoryConnectionStore) GetAll() (connections []connection.IConnection, err error) {
	s.withRead(func() {
		var allConns []connection.IConnection
		for _, c := range s.store {
			allConns = append(allConns, c)
		}
		connections = allConns
	})
	return
}
