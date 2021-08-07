package connection_manager

import (
	"wsdk/common/logger"
	"wsdk/common/utils"
	"wsdk/relay_common/connection"
	"wsdk/relay_server/container"
)

// TODO replace anonymous client manager with this, then change client manager to actual registered client manager

type ConnectionManager struct {
	logger            *logger.SimpleLogger
	connStore         IConnectionStore             // all connection management
	activeClientStore IActiveClientConnectionStore // connected client management
}

type IConnectionManager interface {
	Accept(connection.IConnection) error
	Disconnect(connection.IConnection) error
	DisconnectAllConnections() error
	GetConnectionsByClientId(string) ([]connection.IConnection, error)
	RegisterClientToConnection(clientId string, addr string) error
	// handleClientDowngrade(credential removal)
	// handleConnectionClosed
	// handleConnectionError
}

func (m *ConnectionManager) initNotifications() {
	// TODO on client downgrade event
	// events.OnEvent()
}

func (m *ConnectionManager) Accept(conn connection.IConnection) (err error) {
	logger.LogError(m.logger, "Accept", err)
	err = m.connStore.Add(conn)
	if err != nil {
		return err
	}
	m.acceptConnection(conn)
	return
}

func (m *ConnectionManager) acceptConnection(conn connection.IConnection) {
	conn.OnError(func(err error) {
		m.handleConnectionError(conn, err)
	})
	conn.OnClose(func(err error) {
		m.handleConnectionClosed(conn, err)
	})
}

func (m *ConnectionManager) handleConnectionClosed(conn connection.IConnection, err error) {
	if err == nil {
		m.logger.Printf("connection %s closed", conn.Address())
	} else {
		m.logger.Printf("connection %s closed with error %s", conn.Address(), err.Error())
	}
}

func (m *ConnectionManager) handleConnectionError(conn connection.IConnection, err error) {
	m.logger.Printf("connection %s has encountered an error %s", conn.Address(), err.Error())
}

func (m *ConnectionManager) handleClientDowngrade(clientId string) {
	err := m.activeClientStore.DeleteAll(clientId)
	if err != nil {
		m.logger.Printf("unable to successfully downgrade client %s due to %s", clientId, err.Error())
	}
}

// TODO finish

func init() {
	container.Container.Singleton(func() IConnectionManager {
		// TODO use NewFunc
		return nil
	})
}
