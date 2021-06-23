package relay_server

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"wsdk/relay_common/messages"
)

type ClientManager struct {
	ctx     *Context
	clients map[string]*WRServerClient
	lock    *sync.RWMutex
}

type IClientManager interface {
	HasClient(id string) bool
	GetClient(id string) *WRServerClient
	GetClientByAddr(addr string) *WRServerClient
	WithAllClients(cb func(clients []*WRServerClient))
	DisconnectClient(id string) error
	DisconnectClientByAddr(addr string) error
	DisconnectAllClients() error
	AddClient(client *WRServerClient) error
	HandleClientConnectionClosed(c *WRServerClient, err error)
	HandleClientError(c *WRServerClient, err error)
}

func NewClientManager(ctx *Context) IClientManager {
	return &ClientManager{
		ctx:     ctx,
		clients: make(map[string]*WRServerClient),
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
	m.ctx.NotificationEmitter().Notify(EventClientConnected, messages.NewNotification(EventClientConnected, client.Id()))
	return nil
}

func (m *ClientManager) DisconnectClient(id string) (err error) {
	client := m.GetClient(id)
	if client == nil {
		return NewClientNotConnectedError(id)
	}
	err = client.Close()
	m.withWrite(func() {
		delete(m.clients, id)
	})
	m.ctx.NotificationEmitter().Notify(
		EventClientDisconnected,
		messages.NewMessage(EventClientDisconnected, "WRelayServer", "", "", messages.MessageTypeInternalNotification, ([]byte)(id)),
	)
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

func (m *ClientManager) WithAllClients(cb func(clients []*WRServerClient)) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	var clients []*WRServerClient
	for _, c := range m.clients {
		clients = append(clients, c)
	}
	cb(clients)
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

func (m *ClientManager) HandleClientConnectionClosed(c *WRServerClient, err error) {
	if err == nil {
		// remove client from connection
		m.DisconnectClient(c.Id())
	} else {
		// unexpected closure
		// service should kill all jobs and transit to DeadMode
		m.withWrite(func() {
			delete(m.clients, c.Id())
		})
		m.ctx.NotificationEmitter().Notify(EventClientUnexpectedClosure, messages.NewNotification(EventClientUnexpectedClosure, c.Id()))

	}
}

func (m *ClientManager) HandleClientError(c *WRServerClient, err error) {
	// just log it
	fmt.Println(c, err)
}
