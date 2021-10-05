package connection_manager

import (
	"errors"
	"fmt"
	"wsdk/common/logger"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/messages"
	"wsdk/relay_server/container"
	"wsdk/relay_server/context"
	"wsdk/relay_server/events"
	"wsdk/relay_server/module_base"
)

type ConnectionManagerModule struct {
	*module_base.ModuleBase
	logger            *logger.SimpleLogger
	connStore         IConnectionStore             // all connection management
	activeClientStore IActiveClientConnectionStore // connected client management
}

type IConnectionManagerModule interface {
	AddConnection(connection.IConnection) error
	Disconnect(string) error
	DisconnectAllConnections() error
	GetConnectionByAddress(string) (connection.IConnection, error)
	GetConnectionsByClientId(string) ([]connection.IConnection, error)
	RegisterClientToConnection(clientId string, addr string) error
	WithAllConnections(func(connection.IConnection)) error

	// AddToConnectionGroup(clientId string, conn connection.IConnection, groupId string) error
}

func NewConnectionManagerModule() IConnectionManagerModule {
	manager := &ConnectionManagerModule{
		logger:            context.Ctx.Logger().WithPrefix("[ConnectionManagerModule]"),
		connStore:         NewInMemoryConnectionStore(),
		activeClientStore: NewInMemoryActiveClientConnectionStore(),
	}
	manager.initNotifications()
	return manager
}

func (m *ConnectionManagerModule) Init() error {
	m.ModuleBase = module_base.NewModuleBase("ConnectionManager", func() error {
		var holder IConnectionManagerModule
		m.disposeNotifications()
		return container.Container.RemoveByType(holder)
	})
	m.logger = m.Logger()
	m.connStore = NewInMemoryConnectionStore()
	m.activeClientStore = NewInMemoryActiveClientConnectionStore()
	m.initNotifications()
	return container.Container.Singleton(func() IConnectionManagerModule {
		return m
	})
}

func (m *ConnectionManagerModule) onServerCloseHandler(message messages.IMessage) {
	m.DisconnectAllConnections()
}

func (m *ConnectionManagerModule) initNotifications() {
	// TODO on client downgrade event
	// events.OnEvent()
	events.OnEvent(events.EventServerClosed, m.onServerCloseHandler)
}

func (m *ConnectionManagerModule) disposeNotifications() {
	events.OffEvent(events.EventServerClosed, m.onServerCloseHandler)
}

func (m *ConnectionManagerModule) AddConnection(conn connection.IConnection) (err error) {
	defer logger.LogError(m.logger, "AddConnection", err)
	err = m.connStore.Add(conn)
	if err != nil {
		return err
	}
	m.acceptConnection(conn)
	return
}

func (m *ConnectionManagerModule) acceptConnection(conn connection.IConnection) {
	conn.OnError(func(err error) {
		m.handleConnectionError(conn, err)
	})
	conn.OnClose(func(err error) {
		m.handleConnectionClosed(conn, err)
	})
}

func (m *ConnectionManagerModule) handleConnectionClosed(conn connection.IConnection, err error) {
	m.connStore.Delete(conn.Address())
	if err == nil {
		m.logger.Printf("connection %s closed", conn.Address())
	} else {
		m.logger.Printf("connection %s closed with error %s", conn.Address(), err.Error())
	}
}

func (m *ConnectionManagerModule) handleClientConnectionClosed(clientId string) {
	conns, err := m.activeClientStore.Get(clientId)
	if err != nil {
		m.logger.Printf("unable to handle client connection closure due to %s", err.Error())
		return
	}
	if len(conns) == 0 {
		m.logger.Printf("all connections from client %s is gone", clientId)
		events.EmitEvent(events.EventClientConnectionGone, clientId)
	}
}

func (m *ConnectionManagerModule) handleConnectionError(conn connection.IConnection, err error) {
	m.logger.Printf("connection %s has encountered an error %s, closing connection...", conn.Address(), err.Error())
	conn.Close()
}

func (m *ConnectionManagerModule) handleClientDowngrade(clientId string) {
	err := m.activeClientStore.DeleteAll(clientId)
	if err != nil {
		m.logger.Printf("unable to successfully downgrade client %s due to %s", clientId, err.Error())
	}
}

func (m *ConnectionManagerModule) Disconnect(addr string) error {
	c, err := m.connStore.Get(addr)
	if err != nil {
		return err
	}
	if c == nil {
		return errors.New(fmt.Sprintf("can not find connection by address %s", addr))
	}
	return c.Close()
}

func (m *ConnectionManagerModule) DisconnectAllConnections() (err error) {
	conns, err := m.connStore.GetAll()
	if err != nil {
		return
	}
	for _, c := range conns {
		e := c.Close()
		if e != nil {
			err = e
		}
	}
	return
}

func (m *ConnectionManagerModule) GetConnectionByAddress(address string) (connection.IConnection, error) {
	return m.connStore.Get(address)
}

func (m *ConnectionManagerModule) GetConnectionsByClientId(clientId string) ([]connection.IConnection, error) {
	addrs, err := m.activeClientStore.Get(clientId)
	if err != nil {
		return nil, err
	}
	conns := make([]connection.IConnection, len(addrs), len(addrs))
	for i, a := range addrs {
		conns[i], err = m.connStore.Get(a)
		if err != nil {
			return nil, err
		}
	}
	return conns, nil
}

func (m *ConnectionManagerModule) RegisterClientToConnection(clientId string, addr string) error {
	conn, err := m.connStore.Get(addr)
	if err != nil {
		return err
	}
	if conn == nil {
		return errors.New(fmt.Sprintf("can not find connection by address %s", addr))
	}
	err = m.activeClientStore.Add(clientId, addr)
	if err != nil {
		return err
	}
	conn.OnClose(func(err error) {
		m.handleConnectionClosed(conn, err)
		// delete the clientId-connection record from active client store
		m.activeClientStore.Delete(clientId, addr)
		events.EmitEvent(events.EventClientConnectionClosed, clientId)
		m.handleClientConnectionClosed(clientId)
	})
	events.EmitEvent(events.EventClientConnectionEstablished, clientId)
	return nil
}

func (m *ConnectionManagerModule) WithAllConnections(cb func(iConnection connection.IConnection)) error {
	conns, err := m.connStore.GetAll()
	if err != nil {
		return err
	}
	for _, c := range conns {
		cb(c)
	}
	return nil
}

/*
func (m *ConnectionManagerModule) assembleConnectionGroupId(id string) string {
	return fmt.Sprintf("conn-group-%s", id)
}

func (m *ConnectionManagerModule) isConnGroupId(id string) bool {
	return strings.HasPrefix(id, "conn-group-")
}

func (m *ConnectionManagerModule) AddToConnectionGroup(clientId string, conn connection.IConnection, groupId string) error {
	if m.isConnGroupId(conn.Address()) {
		return errors.New("connection group can not form another connection group")
	}
	groupId = m.assembleConnectionGroupId(groupId)
	group, err := m.connStore.Get(groupId)
	if err != nil {
		return err
	}
	// remove from active client store as we will manage it from connection group
	m.activeClientStore.Delete(clientId, conn.Address())
	if group != nil {
		// group already exist
		group.(connection.IConnectionGroup).Add(conn)
	} else {
		// new group
		group = connection.NewConnectionGroup(groupId, conn)
		m.activeClientStore.Add(clientId, groupId)
	}
	conn.OnClose(func(err error) {
		m.handleConnectionClosed(conn, err)
		// delete the clientId-connection record from conn-group
		group.(connection.IConnectionGroup).Remove(conn.Address())
		events.EmitEvent(events.EventClientConnectionClosed, clientId)
		m.handleClientConnectionClosed(clientId)
	})
	return nil
}
*/

func Load() error {
	return container.Container.Singleton(func() IConnectionManagerModule {
		return NewConnectionManagerModule()
	})
}
