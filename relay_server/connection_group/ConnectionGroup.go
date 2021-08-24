package connection_group

import (
	"sync"
	"time"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/notification"
)

// TODO do we really need this? do we really need the to in the message? do we only need this in pubsub? or do we need this for client connection pool?

type connNode struct {
	connection.IConnection
	prev *connNode
	next *connNode
}

type IConnectionGroup interface {
	connection.IConnection
	Add(connection.IConnection) bool
	Remove(connection.IConnection) bool
}

type ConnectionGroup struct {
	conns map[string]*connNode
	curr  *connNode
	lock  *sync.RWMutex
}

func (g *ConnectionGroup) Address() string {
	panic("implement me")
}

func (g *ConnectionGroup) ReadingLoop() {
	panic("implement me")
}

func (g *ConnectionGroup) Request(message messages.IMessage) (messages.IMessage, error) {
	panic("implement me")
}

func (g *ConnectionGroup) RequestWithTimeout(message messages.IMessage, duration time.Duration) (messages.IMessage, error) {
	panic("implement me")
}

func (g *ConnectionGroup) Send(message messages.IMessage) error {
	panic("implement me")
}

func (g *ConnectionGroup) OnIncomingMessage(f func(message messages.IMessage)) {
	panic("implement me")
}

func (g *ConnectionGroup) OnceMessage(s string, f func(messages.IMessage)) (notification.Disposable, error) {
	panic("implement me")
}

func (g *ConnectionGroup) OnMessage(s string, f func(messages.IMessage)) (notification.Disposable, error) {
	panic("implement me")
}

func (g *ConnectionGroup) OffMessage(s string, f func(messages.IMessage)) {
	panic("implement me")
}

func (g *ConnectionGroup) OffAll(s string) {
	panic("implement me")
}

func (g *ConnectionGroup) OnError(f func(error)) {
	panic("implement me")
}

func (g *ConnectionGroup) OnClose(f func(error)) {
	panic("implement me")
}

func (g *ConnectionGroup) Close() error {
	panic("implement me")
}

func (g *ConnectionGroup) ConnectionType() uint8 {
	panic("implement me")
}

func (g *ConnectionGroup) String() string {
	panic("implement me")
}

func (g *ConnectionGroup) IsLive() bool {
	panic("implement me")
}

func (g *ConnectionGroup) withWrite(cb func()) {
	g.lock.Lock()
	defer g.lock.Unlock()
	cb()
}

func (g *ConnectionGroup) Add(conn connection.IConnection) bool {
	if g.conns[conn.Address()] != nil {
		return false
	}
	g.withWrite(func() {
		node := &connNode{conn, nil, nil}
		g.conns[conn.Address()] = node
		if g.curr == nil {
			g.curr = node
			node.prev = nil
			node.next = node
		} else {
			node.prev = g.curr
			node.next = g.curr.next
			g.curr.next = node
		}
	})
	return true
}

func (g *ConnectionGroup) Remove(conn connection.IConnection) bool {
	node := g.conns[conn.Address()]
	if node == nil {
		return false
	}
	g.withWrite(func() {
		if node.prev != nil {
			node.prev.next = node.next
		}
		if node.next.prev != nil {
			node.next.prev = node.prev
		}
	})
	return true
}

func NewConnectionGroup(conn connection.IConnection) IConnectionGroup {
	return &ConnectionGroup{
		conns: make(map[string]*connNode),
		curr:  nil,
		lock:  new(sync.RWMutex),
	}
}
