package managers

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"wsdk/relay_common/messages"
	"wsdk/relay_server/client"
	"wsdk/relay_server/container"
	"wsdk/relay_server/context"
	servererror "wsdk/relay_server/errors"
	"wsdk/relay_server/events"
)

func init() {
	container.Container.RegisterComponent(ClientManagerId, NewClientManager())
}

const ClientManagerId = "ClientManager"

type ClientManager struct {
	clients map[string]*client.Client
	lock    *sync.RWMutex
}

type IClientManager interface {
	HasClient(id string) bool
	GetClient(id string) *client.Client
	GetClientByAddr(addr string) *client.Client
	WithAllClients(cb func(clients []*client.Client))
	DisconnectClient(id string) error
	DisconnectClientByAddr(addr string) error
	DisconnectAllClients() error
	AddClient(client *client.Client) error
	HandleClientConnectionClosed(c *client.Client, err error)
	HandleClientError(c *client.Client, err error)
}

func NewClientManager() IClientManager {
	return &ClientManager{
		clients: make(map[string]*client.Client),
		lock:    new(sync.RWMutex),
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

func (m *ClientManager) GetClient(id string) *client.Client {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.clients[id]
}

func (m *ClientManager) AddClient(client *client.Client) error {
	id := client.Id()
	if m.HasClient(id) {
		return servererror.NewClientAlreadyConnectedError(id)
	}
	m.withWrite(func() {
		m.clients[id] = client
	})
	context.Ctx.NotificationEmitter().Notify(events.EventClientConnected, messages.NewNotification(events.EventClientConnected, client.Id()))
	return nil
}

func (m *ClientManager) DisconnectClient(id string) (err error) {
	client := m.GetClient(id)
	if client == nil {
		return servererror.NewClientNotConnectedError(id)
	}
	err = client.Close()
	m.withWrite(func() {
		delete(m.clients, id)
	})
	context.Ctx.NotificationEmitter().Notify(
		events.EventClientDisconnected,
		messages.NewMessage(events.EventClientDisconnected, "WRelayServer", "", "", messages.MessageTypeInternalNotification, ([]byte)(id)),
	)
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
		context.Ctx.NotificationEmitter().Notify(events.EventClientUnexpectedClosure, messages.NewNotification(events.EventClientUnexpectedClosure, c.Id()))
	}
}

func (m *ClientManager) HandleClientError(c *client.Client, err error) {
	// just log it
	fmt.Println(c, err)
}
