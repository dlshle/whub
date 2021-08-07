package connection_manager

import (
	"errors"
	"fmt"
	"wsdk/common/data_structures"
)

const (
	DefaultMaxConnectionsPerClient = 15
)

// clinetId: []connectionAddress
type IActiveClientConnectionStore interface {
	Add(string, string) error
	Has(string) (bool, error)
	Delete(string, string) error
	DeleteAll(string) error
	Get(string) ([]string, error)
}

type InMemoryActiveClientConnectionStore struct {
	store map[string]data_structures.ISet
}

func (s *InMemoryActiveClientConnectionStore) getOrCreateAndAdd(clientId string, addr string) bool {
	set := s.store[clientId]
	if set == nil {
		set = data_structures.NewSafeSet()
		s.store[clientId] = set
	}
	return set.Add(addr)
}

func (s *InMemoryActiveClientConnectionStore) Add(clientId string, addr string) error {
	if !s.getOrCreateAndAdd(clientId, addr) {
		return errors.New(fmt.Sprintf("connection %s has already been registered as client %s", addr, clientId))
	}
	return nil
}

func (s *InMemoryActiveClientConnectionStore) Has(clientId string) (bool, error) {
	return s.store[clientId] != nil, nil
}

func (s *InMemoryActiveClientConnectionStore) Get(clientId string) ([]string, error) {
	set := s.store[clientId]
	if set == nil {
		return []string{}, nil
	}
	allData := set.GetAll()
	connections := make([]string, len(allData), len(allData))
	for i := range allData {
		connections[i] = allData[i].(string)
	}
	return connections, nil
}

func (s *InMemoryActiveClientConnectionStore) Delete(clientId string, addr string) error {
	set := s.store[clientId]
	if set == nil {
		return errors.New(fmt.Sprintf("client %s does not have connection yet", clientId))
	}
	if !set.Delete(addr) {
		return errors.New(fmt.Sprintf("client %s does not have connection %s registered", clientId, addr))
	}
	if set.Size() == 0 {
		delete(s.store, clientId)
	}
	return nil
}

func (s *InMemoryActiveClientConnectionStore) DeleteAll(clientId string) error {
	set := s.store[clientId]
	if set == nil {
		return errors.New(fmt.Sprintf("client %s does not have connection yet", clientId))
	}
	set.Clear()
	delete(s.store, clientId)
	return nil
}
