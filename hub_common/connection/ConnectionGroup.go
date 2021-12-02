package connection

import (
	"errors"
	"fmt"
	"sync"
	"time"
	"whub/hub_common/messages"
	"whub/hub_common/notification"
)

type connNode struct {
	IConnection
	prev *connNode
	next *connNode
}

type IConnectionGroup interface {
	IConnection
	Add(IConnection) bool
	Remove(string) bool
}

type ConnectionGroup struct {
	id    string
	conns map[string]*connNode
	curr  *connNode
	lock  *sync.RWMutex
}

func (g *ConnectionGroup) withWrite(cb func()) {
	g.lock.Lock()
	defer g.lock.Unlock()
	cb()
}

func (g *ConnectionGroup) withRead(cb func()) {
	g.lock.RLock()
	defer g.lock.RUnlock()
	cb()
}

func (g *ConnectionGroup) Add(conn IConnection) bool {
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

func (g *ConnectionGroup) Remove(addr string) bool {
	node := g.conns[addr]
	if node == nil {
		return false
	}
	g.withWrite(func() {
		delete(g.conns, addr)
		if g.curr == node {
			g.curr = g.curr.next
		}
		if node.prev != nil {
			node.prev.next = node.next
		}
		if node.next.prev != nil {
			node.next.prev = node.prev
		}
		if len(g.conns) == 0 {
			// TODO what to do when deleting the last node
		}
	})
	return true
}

func (g *ConnectionGroup) nextConn() IConnection {
	node := g.curr
	g.curr = g.curr.next
	return node
}

func (g *ConnectionGroup) Address() string {
	return fmt.Sprintf("conn-group-%s", g.id)
}

func (g *ConnectionGroup) ReadingLoop() {
}

func (g *ConnectionGroup) Request(message messages.IMessage) (messages.IMessage, error) {
	next := g.nextConn()
	if next == nil {
		return nil, errors.New("no valid connection")
	}
	return next.Request(message)
}

func (g *ConnectionGroup) RequestWithTimeout(message messages.IMessage, duration time.Duration) (messages.IMessage, error) {
	next := g.nextConn()
	if next == nil {
		return nil, errors.New("no valid connection")
	}
	return next.RequestWithTimeout(message, duration)
}

func (g *ConnectionGroup) Send(message messages.IMessage) error {
	next := g.nextConn()
	if next == nil {
		return errors.New("no valid connection")
	}
	return next.Send(message)
}

func (g *ConnectionGroup) withEachConn(cb func(IConnection)) {
	g.withRead(func() {
		for _, n := range g.conns {
			cb(n)
		}
	})
}

func (g *ConnectionGroup) OnIncomingMessage(f func(message messages.IMessage)) {
	g.withEachConn(func(conn IConnection) {
		conn.OnIncomingMessage(f)
	})
}

func (g *ConnectionGroup) OnceMessage(s string, f func(messages.IMessage)) (notification.Disposable, error) {
	return nil, nil
}

func (g *ConnectionGroup) OnMessage(s string, f func(messages.IMessage)) (notification.Disposable, error) {
	return nil, nil
}

func (g *ConnectionGroup) OffMessage(s string, f func(messages.IMessage)) {
}

func (g *ConnectionGroup) OffAll(s string) {
}

func (g *ConnectionGroup) OnError(f func(error)) {
	g.withEachConn(func(conn IConnection) {
		conn.OnError(f)
	})
}

func (g *ConnectionGroup) OnClose(f func(error)) {
	g.withEachConn(func(conn IConnection) {
		conn.OnClose(f)
	})
}

func (g *ConnectionGroup) Close() (err error) {
	g.withEachConn(func(conn IConnection) {
		err = conn.Close()
	})
	return err
}

func (g *ConnectionGroup) ConnectionType() uint8 {
	return g.curr.ConnectionType()
}

func (g *ConnectionGroup) String() string {
	return g.Address()
}

func (g *ConnectionGroup) IsLive() bool {
	return g.curr.IsLive()
}

func NewConnectionGroup(id string, conn IConnection) IConnectionGroup {
	group := &ConnectionGroup{
		id:    id,
		conns: make(map[string]*connNode),
		curr:  nil,
		lock:  new(sync.RWMutex),
	}
	group.Add(conn)
	return group
}
