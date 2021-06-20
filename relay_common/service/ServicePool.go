package service

import (
	"errors"
	"fmt"
	"sync"
	"wsdk/common/async"
	"wsdk/relay_common"
)

const (
	MinServicePoolSize = 128
	MaxServicePoolSize = 2048
)

type IServiceTaskQueue interface {
	Get(id string) *ServiceRequest
	Start()
	Stop()
	Add(message *ServiceRequest) bool
	Remove(id string) bool
	Has(id string) bool
	KillAll() error
	Cancel(id string) error
	CancelAll() error
	Size() int
}

type ServiceTaskQueue struct {
	pool       *async.AsyncPool
	executor   relay_common.IRequestExecutor
	messageSet map[string]*ServiceRequest
	lock       *sync.RWMutex
}

func NewServiceTaskQueue(executor relay_common.IRequestExecutor, pool *async.AsyncPool) *ServiceTaskQueue {
	return &ServiceTaskQueue{pool, executor, make(map[string]*ServiceRequest), new(sync.RWMutex)}
}

func (p *ServiceTaskQueue) withWrite(cb func()) {
	p.lock.Lock()
	defer p.lock.Unlock()
	cb()
}

func (p *ServiceTaskQueue) withAll(operation func(message *ServiceRequest) error) error {
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

func (p *ServiceTaskQueue) Start() {
	p.pool.Start()
}

func (p *ServiceTaskQueue) Stop() {
	p.pool.Stop()
}

func (p *ServiceTaskQueue) Has(id string) bool {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.messageSet[id] != nil
}

func (p *ServiceTaskQueue) Get(id string) *ServiceRequest {
	if !p.Has(id) {
		return nil
	}
	return p.messageSet[id]
}

func (p *ServiceTaskQueue) Remove(id string) bool {
	if !p.Has(id) {
		return false
	}
	p.withWrite(func() {
		delete(p.messageSet, id)
	})
	return true
}

func (p *ServiceTaskQueue) Add(message *ServiceRequest) bool {
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

func (p *ServiceTaskQueue) KillAll() (errMsg error) {
	return p.withAll(func(message *ServiceRequest) error {
		return message.Kill()
	})
}

func (p *ServiceTaskQueue) Cancel(id string) error {
	msg := p.Get(id)
	if msg == nil {
		return errors.New("Can not find messages " + id + " from the set")
	}
	return msg.Cancel()
}

func (p *ServiceTaskQueue) CancelAll() error {
	return p.withAll(func(message *ServiceRequest) error {
		return message.Cancel()
	})
}

func (p *ServiceTaskQueue) Size() int {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return len(p.messageSet)
}
