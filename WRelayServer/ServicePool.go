package WRelayServer

import (
	"errors"
	"fmt"
	"github.com/dlshle/gommon/async"
	"runtime"
	"sync"
	"wsdk/WRCommon"
)

const DefaultServicePoolSize = 2048

type IServicePool interface {
	Get(id string) *WRCommon.ServiceMessage
	Start()
	Stop()
	Add(message *WRCommon.ServiceMessage) bool
	Remove(id string) bool
	Has(id string) bool
	Pull(id string) *WRCommon.ServiceMessage // get and remove
	KillAll() error
	Cancel(id string) error
	CancelAll() error
	Size() int
}

type ServicePool struct {
	pool       *async.AsyncPool
	executor   WRCommon.IRequestExecutor
	messageSet map[string]*WRCommon.ServiceMessage
	lock       *sync.RWMutex
}

func NewServicePool(executor WRCommon.IRequestExecutor, size int) *ServicePool {
	return &ServicePool{async.NewAsyncPool("[ServicePool]", size, runtime.NumCPU() * 4), executor, make(map[string]*WRCommon.ServiceMessage), new(sync.RWMutex)}
}

func (p *ServicePool) withWrite(cb func()) {
	p.lock.Lock()
	defer p.lock.Unlock()
	cb()
}

func (p *ServicePool) withAll(operation func(message *WRCommon.ServiceMessage) error) error {
	errorMessage := ""
	hasError := false
	p.withWrite(func() {
		for _, v := range p.messageSet {
			err := operation(v)
			if err != nil {
				hasError = true
				fmt.Sprintf("%s%s", errorMessage, err)
			}
		}
	})
	if hasError {
		return errors.New(errorMessage)
	}
	return nil
}

func (p *ServicePool) Start() {
	p.pool.Start()
}

func (p *ServicePool) Stop() {
	p.pool.Stop()
}

func (p *ServicePool) Has(id string) bool {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.messageSet[id] != nil
}

func (p *ServicePool) Get(id string) *WRCommon.ServiceMessage {
	if !p.Has(id) {
		return nil
	}
	return p.messageSet[id]
}

func (p *ServicePool) Remove(id string) bool {
	if !p.Has(id) {
		return false
	}
	p.withWrite(func() {
		delete(p.messageSet, id)
	})
	return true
}

func (p *ServicePool) Add(message *WRCommon.ServiceMessage) bool {
	if p.Has(message.Id()) {
		return false
	}
	p.withWrite(func() {
		p.messageSet[message.Id()] = message
	})
	p.pool.Schedule(func() {
		// execute should take care of the execution logic
		p.executor.Execute(message)
		p.Remove(message.Id())
	})
	return true
}

func (p *ServicePool) Pull(id string) (msg *WRCommon.ServiceMessage) {
	msg = p.Get(id)
	if msg == nil {
		return nil
	}
	p.Remove(id)
	return
}

func (p *ServicePool) KillAll() (errMsg error) {
	return p.withAll(func(message *WRCommon.ServiceMessage) error {
		return message.Kill()
	})
}

func (p *ServicePool) Cancel(id string) error {
	msg := p.Get(id)
	if msg == nil {
		return errors.New("Can not find message " + id + " from the set")
	}
	return msg.Cancel()
}

func (p *ServicePool) CancelAll() error {
	return p.withAll(func(message *WRCommon.ServiceMessage) error {
		return message.Cancel()
	})
}

func (p *ServicePool) Size() int {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return len(p.messageSet)
}