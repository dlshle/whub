package controllers

import (
	"errors"
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
	container.Container.Singleton(func() IClientManager { return NewClientManager() })
}

const ClientManagerId = "ClientManager"

type ClientManager struct {
	clients map[string]*client.Client
	lock    *sync.RWMutex
	logger  *logger.SimpleLogger
}

type IClientManager interface {
	HasClient(id string) bool
	GetClient(id string) *client.Client
	GetClientByAddr(addr string) *client.Client
	WithAllClients(cb func(clients []*client.Client))
	DisconnectClient(id string) error
	DisconnectClientByAddr(addr string) error
	DisconnectAllClients() error
	AcceptClient(id string, client *client.Client) error
	HandleClientConnectionClosed(c *client.Client, err error)
	HandleClientError(c *client.Client, err error)
}

func NewClientManager() IClientManager {
	manager := &ClientManager{
		clients: make(map[string]*client.Client),
		lock:    new(sync.RWMutex),
		logger:  context.Ctx.Logger().WithPrefix("[ClientManager]"),
	}
	manager.initNotificationHandlers()
	return manager
}

func (m *ClientManager) initNotificationHandlers() {
	events.OnEvent(events.EventServerClosed, func(msg *messages.Message) {
		m.logger.Println("received ServerClosed event, disconnecting all clients...")
		m.DisconnectAllClients()
	})
}

func (m *ClientManager) withWrite(cb func()) {
	m.lock.Lock()
	defer m.lock.Unlock()
	cb()
}

func (m *ClientManager) HasClient(id string) bool {
	return m.GetClient(id) != nil
}

func (m *ClientManager) GetClient(id string) *client.Client {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.clients[id]
}

func (m *ClientManager) AcceptClient(id string, client *client.Client) error {
	if m.HasClient(id) {
		return servererror.NewClientAlreadyConnectedError(id)
	}
	m.withWrite(func() {
		m.clients[id] = client
	})
	m.handleClientAccepted(client)
	m.logger.Printf("client (%s, %s) has been accepted %v", id, client.Address(), client.Describe())
	return nil
}

func (m *ClientManager) handleClientAccepted(client *client.Client) {
	m.initClientCallbackHandlers(client)
	events.EmitEvent(events.EventClientConnected, client.Id())
}

func (m *ClientManager) initClientCallbackHandlers(client *client.Client) {
	client.OnClose(func(err error) {
		m.HandleClientConnectionClosed(client, err)
	})
	client.OnError(func(err error) {
		m.HandleClientError(client, err)
	})
}

func (m *ClientManager) DisconnectClient(id string) (err error) {
	client := m.GetClient(id)
	defer m.logger.Printf("error while disconnecting client %s due to %v", id, err)
	if client == nil {
		err = servererror.NewClientNotConnectedError(id)
		return
	}
	err = client.Close()
	m.withWrite(func() {
		delete(m.clients, id)
	})
	events.EmitEvent(events.EventClientDisconnected, id)
	return
}

func (m *ClientManager) findClientByAddr(addr string) *client.Client {
	m.lock.RLock()
	defer m.lock.RUnlock()
	for _, c := range m.clients {
		if c.Address() == addr {
			return c
		}
	}
	return nil
}

func (m *ClientManager) GetClientByAddr(addr string) *client.Client {
	return m.findClientByAddr(addr)
}

func (m *ClientManager) WithAllClients(cb func(clients []*client.Client)) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	var clients []*client.Client
	for _, c := range m.clients {
		clients = append(clients, c)
	}
	cb(clients)
}

func (m *ClientManager) DisconnectClientByAddr(addr string) error {
	client := m.GetClientByAddr(addr)
	if client == nil {
		return servererror.NewCanNotFindClientByAddr(addr)
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

func (m *ClientManager) HandleClientConnectionClosed(c *client.Client, err error) {
	if err == nil {
		// remove client from connection
		m.DisconnectClient(c.Id())
	} else {
		// unexpected closure
		// service should kill all jobs and transit to DeadMode
		m.withWrite(func() {
			delete(m.clients, c.Id())
		})
		events.EmitEvent(events.EventClientUnexpectedClosure, c.Id())
	}
}

func (m *ClientManager) HandleClientError(c *client.Client, err error) {
	// just log it
	m.logger.Printf("client (%s, %s) on connection error %s", c.Id(), c.Address(), err.Error())
}
