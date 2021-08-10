package service

import (
	"errors"
	"fmt"
	"sync"
	"wsdk/common/async"
	"wsdk/relay_common/messages"
)

const (
	MinServicePoolSize = 128
	MaxServicePoolSize = 2048
)

type IServiceTaskQueue interface {
	Get(id string) *ServiceRequest
	Stop()
	Schedule(message *ServiceRequest) *async.WaitLock
	Remove(id string) bool
	Has(id string) bool
	KillAll() error
	Cancel(id string) error
	CancelAll() error
	Size() int
}

type ServiceTaskQueue struct {
	hostId     string
	pool       async.IAsyncPool
	executor   IRequestExecutor
	messageSet map[string]*ServiceRequest
	lock       *sync.RWMutex
}

func NewServiceTaskQueue(hostId string, executor IRequestExecutor, pool async.IAsyncPool) *ServiceTaskQueue {
	return &ServiceTaskQueue{hostId, pool, executor, make(map[string]*ServiceRequest), new(sync.RWMutex)}
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

func (p *ServiceTaskQueue) Schedule(request *ServiceRequest) *async.WaitLock {
	if p.Has(request.Id()) {
		return nil
	}
	p.withWrite(func() {
		p.messageSet[request.Id()] = request
	})
	return p.pool.Schedule(func() {
		// check if message_dispatcher is processable
		if UnProcessableServiceRequestMap[request.Status()] {
			request.Resolve(messages.NewErrorMessage(request.Id(), p.hostId, request.From(), request.Uri(), "request has been cancelled or target server is dead"))
			return
		}
		request.TransitStatus(ServiceRequestStatusProcessing)
		// execute should take care of the execution logic
		p.executor.Execute(request)
		p.Remove(request.Id())
	})
}

func (p *ServiceTaskQueue) KillAll() (errMsg error) {
	return p.withAll(func(message *ServiceRequest) error {
		return message.Kill()
	})
}

func (p *ServiceTaskQueue) Cancel(id string) error {
	msg := p.Get(id)
	if msg == nil {
		return errors.New("Can not find message_dispatcher " + id + " from the set")
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
