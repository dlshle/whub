package connections

// TODO use this to manage client connections
import (
	"context"
	"errors"
	"wsdk/common/logger"
	client_ctx "wsdk/relay_client/context"
	"wsdk/relay_common/connection"
)

type IConnectionManager interface {
	Start() error
	Connection() (connection.IConnection, error)
	ForEachConnection(cb func(connection.IConnection))
	SetOnConnected(cb func(connection.IConnection))
	SetOnError(cb func(err error))
	Close()
}

type ConnectionManager struct {
	ctx             context.Context
	factory         func() (connection.IConnection, error)
	cancelFunc      func()
	onConnected     func(connection.IConnection)
	onProducerError func(error)
	consumerQueue   chan connection.IConnection
	producerQueue   chan bool // size = consumerQueue.length - 1
	logger          *logger.SimpleLogger
}

func NewConnectionManager(factory func() (connection.IConnection, error), inUseConnections int) IConnectionManager {
	manager := &ConnectionManager{
		ctx:           client_ctx.Ctx.Context(),
		cancelFunc:    client_ctx.Ctx.Stop,
		factory:       factory,
		consumerQueue: make(chan connection.IConnection, inUseConnections),
		producerQueue: make(chan bool, inUseConnections-1),
		logger:        client_ctx.Ctx.Logger().WithPrefix("[ConnectionManager]"),
		onConnected:   func(connection.IConnection) {},
	}
	for i := 0; i < inUseConnections-2; i++ {
		manager.producerQueue <- true
	}
	return manager
}

func (m *ConnectionManager) Start() error {
	if err := m.produce(); err != nil {
		return err
	}
	client_ctx.Ctx.AsyncTaskPool().Schedule(m.producer)
	return nil
}

func (m *ConnectionManager) producer() {
	select {
	case <-m.ctx.Done():
		return
	case <-m.producerQueue:
		if err := m.produce(); err != nil {
			return
		}
	}
}

func (m *ConnectionManager) produce() error {
	conn, err := m.tryToCreateConnection(3, nil)
	if err != nil {
		m.onProducerError(err)
		return err
	}
	m.consumerQueue <- conn
	return nil
}

func (m *ConnectionManager) tryToCreateConnection(retryCount int, lastErr error) (connection.IConnection, error) {
	if retryCount == 0 {
		return nil, lastErr
	}
	conn, err := m.factory()
	if err != nil {
		return m.tryToCreateConnection(retryCount-1, err)
	}
	m.onConnected(conn)
	return conn, nil
}

func (m *ConnectionManager) Connection() (connection.IConnection, error) {
	return m.nextConnection()
}

func (m *ConnectionManager) nextConnection() (connection.IConnection, error) {
	select {
	case <-m.ctx.Done():
		break
	default:
		for len(m.consumerQueue) > 0 {
			conn := <-m.consumerQueue
			if conn.IsLive() {
				// put this conn to the back of the queue
				m.consumerQueue <- conn
				return conn, nil
			}
			m.producerQueue <- true
		}
	}
	// no conn works or closed
	return nil, errors.New("connection error")
}

func (m *ConnectionManager) Close() {
	m.handleClose()
}

func (m *ConnectionManager) handleError(err error) {
	// log error
	m.logger.Println("connection manager error: ", err.Error())
	if m.onProducerError != nil {
		m.onProducerError(err)
	}
	m.handleClose()
}

func (m *ConnectionManager) handleClose() {
	m.cancelFunc()
	for conn := range m.consumerQueue {
		conn.Close()
	}
	close(m.consumerQueue)
}

func (m *ConnectionManager) SetOnConnected(cb func(iConnection connection.IConnection)) {
	m.onConnected = cb
}

func (m *ConnectionManager) SetOnError(cb func(err error)) {
	m.onProducerError = cb
}

func (m *ConnectionManager) ForEachConnection(cb func(connection.IConnection)) {
	for conn := range m.consumerQueue {
		cb(conn)
	}
}
