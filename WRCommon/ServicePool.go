package WRCommon

import (
	"errors"
	"fmt"
	"github.com/dlshle/gommon/async"
	"runtime"
	"sync"
)

type IServicePool interface {
	Get(id string) *ServiceMessage
	Start() error
	Stop() error
	Add(message *ServiceMessage) bool
	Remove(id string) bool
	Has(id string) bool
	Pull(id string) *ServiceMessage // get and remove
	KillAll() error
	Cancel(id string) error
	CancelAll() error
	Size() int
}

type ServicePool struct {
	pool       *async.AsyncPool
	executor   IRequestExecutor
	messageSet map[string]*ServiceMessage
	lock       *sync.RWMutex
}

func NewServicePool(executor IRequestExecutor, size int) *ServicePool {
	return &ServicePool{async.NewAsyncPool("[ServicePool]", size, runtime.NumCPU() * 4), executor, make(map[string]*ServiceMessage), new(sync.RWMutex)}
}

func (p *ServicePool) withWrite(cb func()) {
	p.lock.Lock()
	defer p.lock.Unlock()
	cb()
}

func (p *ServicePool) withAll(operation func(message *ServiceMessage) error) error {
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

func (p *ServicePool) Has(id string) bool {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.messageSet[id] != nil
}

func (p *ServicePool) Get(id string) *ServiceMessage {
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

func (p *ServicePool) Add(message *ServiceMessage) bool {
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

func (p *ServicePool) Pull(id string) (msg *ServiceMessage) {
	msg = p.Get(id)
	if msg == nil {
		return nil
	}
	p.Remove(id)
	return
}

func (p *ServicePool) KillAll() (errMsg error) {
	return p.withAll(func(message *ServiceMessage) error {
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
	return p.withAll(func(message *ServiceMessage) error {
		return message.Cancel()
	})
}

func (p *ServicePool) Size() int {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return len(p.messageSet)
}