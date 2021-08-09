package connection

import (
	"sync"
)

const (
	ConnectionPoolErrExceedMaxPoolSize = 0
	ConnectionPoolErrInitConnection    = 1
)

type ConnectionPoolError struct {
	code uint8
	msg  string
}

func NewConnectionPoolError(code uint8, msg string) ConnectionPoolError {
	return ConnectionPoolError{code, msg}
}

func (e ConnectionPoolError) Error() string {
	return e.msg
}

func (e ConnectionPoolError) Code() uint8 {
	return e.code
}

type ConnectionPool struct {
	pool         []IConnection
	currentSize  int
	maxPoolSize  int
	rwLock       *sync.RWMutex
	connInitFunc func() (IConnection, error)
}

type IConnectionPool interface {
	Get() (IConnection, error)
	Dispose() error
}

func (p *ConnectionPool) withWrite(cb func()) {
	p.rwLock.Lock()
	defer p.rwLock.RUnlock()
	cb()
}

func (p *ConnectionPool) withRead(cb func()) {
	p.rwLock.RLock()
	defer p.rwLock.RUnlock()
	cb()
}

// TODO: learn from HikariCP, how to design a good connection pool?
