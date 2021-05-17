package relay_server

import (
	"errors"
	"strings"
	"sync"
)

type ClientManager struct {
	clients map[string]*WRServerClient
	lock *sync.RWMutex
}

type IClientManager interface {
	HasClient(id string) bool
	GetClient(id string) *WRServerClient
	GetClientByAddr(addr string) *WRServerClient
	DisconnectClient(id string) error
	DisconnectClientByAddr(addr string) error
	DisconnectAllClients() error
	AddClient(client *WRServerClient) error
}

func NewClientManager() IClientManager {
	return &ClientManager{
		clients: make(map[string]*WRServerClient),
		lock: new(sync.RWMutex),
	}
}

func (m *ClientManager) withWrite(cb func()) {
	m.lock.Lock()
	defer m.lock.Unlock()
	cb()
}

func (m *ClientManager) HasClient(id string) bool {
	return m.GetClient(id) != nil
}

func (m *ClientManager) GetClient(id string) *WRServerClient {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.clients[id]
}

func (m *ClientManager) AddClient(client *WRServerClient) error {
	id := client.Id()
	if m.HasClient(id) {
		return NewClientAlreadyConnectedError(id)
	}
	m.withWrite(func() {
		m.clients[id] = client
	})
	return nil
}

func (m *ClientManager) DisconnectClient(id string) (err error) {
	client := m.GetClient(id)
	if client == nil {
		return NewClientNotConnectedError(id)
	}
	err = client.Close()
	if err == nil {
		m.withWrite(func() {
			delete(m.clients, id)
		})
	}
	return
}

func (m *ClientManager) findClientByAddr(addr string) *WRServerClient {
	m.lock.RLock()
	defer m.lock.RUnlock()
	for _, c := range m.clients {
		if c.Address() == addr {
			return c
		}
	}
	return nil
}

func (m *ClientManager) GetClientByAddr(addr string) *WRServerClient {
	return m.findClientByAddr(addr)
}

func (m *ClientManager) DisconnectClientByAddr(addr string) error {
	client := m.GetClientByAddr(addr)
	if client == nil {
		return NewCanNotFindClientByAddr(addr)
	}
	return m.DisconnectClient(client.Id())
}

func (m *ClientManager) DisconnectAllClients() error {
	errMsgBuilder := strings.Builder{}
	m.withWrite(func() {
		for _, c := range m.clients {
			errMsgBuilder.WriteString(c.Close().Error() + "\n")
		}
	})
	return errors.New(errMsgBuilder.String())
}
