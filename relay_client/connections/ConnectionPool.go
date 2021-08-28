package connections

// TODO use this to manage client connections
import (
	"context"
	"errors"
	"sync/atomic"
	"wsdk/common/logger"
	"wsdk/relay_client/container"
	client_ctx "wsdk/relay_client/context"
	"wsdk/relay_common/connection"
)

type IConnectionPool interface {
	Start() error
	Get() (connection.IConnection, error)
	Put(connection.IConnection)
	SetOnConnected(cb func(connection.IConnection))
	SetOnError(cb func(err error))
	Close()
}

type ConnectionPool struct {
	ctx                context.Context
	factory            func() (connection.IConnection, error)
	cancelFunc         func()
	onConnected        func(connection.IConnection)
	onProducerError    func(error)
	consumerQueue      chan connection.IConnection
	producerQueue      chan bool // size = consumerQueue.length
	maxServiceConnSize int
	closed             atomic.Value
	logger             *logger.SimpleLogger
}

func NewConnectionPool(factory func() (connection.IConnection, error), numActiveConns int) IConnectionPool {
	if numActiveConns < 3 {
		numActiveConns = 3
	}
	var closed atomic.Value
	closed.Store(false)
	pool := &ConnectionPool{
		ctx:                client_ctx.Ctx.Context(),
		cancelFunc:         client_ctx.Ctx.Stop,
		factory:            factory,
		consumerQueue:      make(chan connection.IConnection, numActiveConns),
		producerQueue:      make(chan bool, numActiveConns),
		maxServiceConnSize: numActiveConns - 1,
		logger:             client_ctx.Ctx.Logger().WithPrefix("[ConnectionPool]"),
		onConnected:        func(connection.IConnection) {},
		closed:             closed,
	}
	for i := 0; i < numActiveConns; i++ {
		pool.producerQueue <- true
	}
	// make it available from the container
	pool.initContainerRegistries()
	return pool
}

func (m *ConnectionPool) initContainerRegistries() {
	container.Container.Singleton(func() IConnectionPool {
		return m
	})
}

func (m *ConnectionPool) Start() error {
	if m.closed.Load().(bool) {
		return errors.New("manager already closed")
	}
	if err := m.produce(); err != nil {
		return err
	}
	client_ctx.Ctx.AsyncTaskPool().Schedule(m.producer)
	return nil
}

func (m *ConnectionPool) producer() {
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-m.producerQueue:
			if err := m.produce(); err != nil {
				return
			}
		}
	}
}

func (m *ConnectionPool) produce() error {
	conn, err := m.tryToCreateConnection(3, nil)
	if err != nil {
		m.onProducerError(err)
		m.handleError(err)
		return err
	}
	m.consumerQueue <- conn
	return nil
}

func (m *ConnectionPool) tryToCreateConnection(retryCount int, lastErr error) (connection.IConnection, error) {
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

func (m *ConnectionPool) Get() (connection.IConnection, error) {
	for conn := range m.consumerQueue {
		if conn.IsLive() {
			return conn, nil
		}
	}
	return nil, errors.New("unable to produce new connection")
}

func (m *ConnectionPool) Put(conn connection.IConnection) {
	if !conn.IsLive() {
		conn.Close()
		m.producerQueue <- true
	} else {
		m.consumerQueue <- conn
	}
}

func (m *ConnectionPool) Close() {
	m.handleClose()
}

func (m *ConnectionPool) handleError(err error) {
	// log error
	m.logger.Println("connection manager error: ", err.Error())
	if m.onProducerError != nil {
		m.onProducerError(err)
	}
	m.handleClose()
}

func (m *ConnectionPool) handleClose() {
	if m.closed.Load().(bool) {
		return
	}
	m.closed.Store(true)
	m.cancelFunc()
	for len(m.consumerQueue) > 0 {
		(<-m.consumerQueue).Close()
	}
	for len(m.producerQueue) > 0 {
		<-m.producerQueue
	}
	if m.closed.Load().(bool) {
		return
	}
	close(m.consumerQueue)
	close(m.producerQueue)
}

func (m *ConnectionPool) SetOnConnected(cb func(iConnection connection.IConnection)) {
	m.onConnected = cb
}

func (m *ConnectionPool) SetOnError(cb func(err error)) {
	m.onProducerError = cb
}
