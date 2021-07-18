package managers

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"wsdk/common/logger"
	"wsdk/relay_common/messages"
	"wsdk/relay_server/client"
	"wsdk/relay_server/container"
	"wsdk/relay_server/context"
	servererror "wsdk/relay_server/errors"
	"wsdk/relay_server/events"
)

func init() {
	container.Container.Singleton(func() IAnonymousClientManager { return NewAnonymousClientManager() })
}

const AnonymousClientManagerId = "AnonymousClientManager"

type AnonymousClientManager struct {
	clientMap map[string]*client.Client
	lock      *sync.RWMutex
	logger    *logger.SimpleLogger
}

type IAnonymousClientManager interface {
	HasClient(id string) bool
	GetClient(id string) *client.Client
	WithAllClients(cb func(clients []*client.Client))
	DisconnectClient(id string) error
	DisconnectClientByAddr(addr string) error
	DisconnectAllClients() error
	AcceptClient(id string, client *client.Client) error
	HandleClientConnectionClosed(c *client.Client, err error)
	HandleClientError(c *client.Client, err error)
	RemoveClient(id string) bool
}

func NewAnonymousClientManager() IAnonymousClientManager {
	m := &AnonymousClientManager{
		clientMap: make(map[string]*client.Client),
		lock:      new(sync.RWMutex),
		logger:    context.Ctx.Logger().WithPrefix("[AnonymousClientManager]"),
	}
	m.initNotificationHandlers()
	return m
}

func (m *AnonymousClientManager) initNotificationHandlers() {
	events.OnEvent(events.EventServerClosed, func(msg *messages.Message) {
		m.logger.Println("ServerClosed event received, disconnecting all clients...")
		err := m.DisconnectAllClients()
		if err != nil {
			m.logger.Println("error while disconnecting all clients: ", err.Error())
		}
	})
}

func (m *AnonymousClientManager) withWrite(cb func()) {
	m.lock.Lock()
	defer m.lock.Unlock()
	cb()
}

func (m *AnonymousClientManager) HasClient(id string) bool {
	return m.GetClient(id) != nil
}

func (m *AnonymousClientManager) GetClient(id string) *client.Client {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.clientMap[id]
}

func (m *AnonymousClientManager) getAllClients() []*client.Client {
	m.lock.RLock()
	defer m.lock.RUnlock()
	clients := make([]*client.Client, 0, len(m.clientMap))
	for _, c := range m.clientMap {
		clients = append(clients, c)
	}
	return clients
}

func (m *AnonymousClientManager) WithAllClients(cb func(clients []*client.Client)) {
	cb(m.getAllClients())
}

func (m *AnonymousClientManager) DisconnectClient(id string) (err error) {
	client := m.GetClient(id)
	defer m.logger.Println("error while disconnecting %s due to: ", err.Error())
	if client == nil {
		err = errors.New(fmt.Sprintf("can not find anonymous client by id %s", id))
		return
	}
	err = client.Close()
	m.withWrite(func() {
		delete(m.clientMap, id)
	})
	return
}

func (m *AnonymousClientManager) DisconnectClientByAddr(addr string) error {
	var errMsg strings.Builder
	m.WithAllClients(func(clients []*client.Client) {
		for _, c := range clients {
			if c.Address() == addr {
				errMsg.WriteString(c.Close().Error())
			}
		}
	})
	return errors.New(errMsg.String())
}

func (m *AnonymousClientManager) DisconnectAllClients() error {
	var errMsg strings.Builder
	m.WithAllClients(func(clients []*client.Client) {
		for _, c := range clients {
			errMsg.WriteString(c.Close().Error())
		}
	})
	return errors.New(errMsg.String())
}

func (m *AnonymousClientManager) AcceptClient(id string, client *client.Client) error {
	if m.HasClient(id) {
		return servererror.NewClientAlreadyConnectedError(id)
	}
	m.withWrite(func() {
		m.clientMap[id] = client
	})
	m.handleClientAccepted(client)
	m.logger.Printf("Anonymous client (%s, %s) has been accepted", id, client.Address())
	return nil
}

func (m *AnonymousClientManager) handleClientAccepted(client *client.Client) {
	client.OnClose(func(err error) {
		m.HandleClientConnectionClosed(client, err)
	})
	client.OnError(func(err error) {
		m.HandleClientError(client, err)
	})
}

func (m *AnonymousClientManager) HandleClientConnectionClosed(c *client.Client, err error) {
	m.logger.Printf("anonymous client connection(%s) closed, err: %v", c.Address(), err)
	m.RemoveClient(c.Address())
}

func (m *AnonymousClientManager) HandleClientError(c *client.Client, err error) {
	// log err
	m.logger.Printf("client %s on connection error %s", c.Address(), err.Error())
}

func (m *AnonymousClientManager) RemoveClient(id string) bool {
	if !m.HasClient(id) {
		return false
	}
	m.withWrite(func() {
		delete(m.clientMap, id)
	})
	m.logger.Printf("client %s has been removed", id)
	return true
}
