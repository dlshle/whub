package service

import (
	"sync"
	"wsdk/relay_common/uri"
)

// ServiceHandler handles service requests
type ServiceHandler struct {
	// TODO trieTree is not good enough, we need something that takes a callback that takes queryParams and pathParams and then execute
	trieTree *uri.TrieTree
	lock     *sync.RWMutex
}

type IServiceHandler interface {
	SupportsUri(uri string) bool
	Register(uri string, handler RequestHandler) bool
	Unregister(uri string) bool
	GetHandler(uri string) RequestHandler
}

func NewServiceHandler() IServiceHandler {
	return &ServiceHandler{
		trieTree: uri.NewTrieTree(),
		lock:     new(sync.RWMutex),
	}
}

func (m *ServiceHandler) withWrite(cb func()) {
	m.lock.Lock()
	defer m.lock.Unlock()
	cb()
}

func (m *ServiceHandler) SupportsUri(internalUri string) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.trieTree.SupportsUri(internalUri)
}

func (m *ServiceHandler) Register(internalUri string, handler RequestHandler) bool {
	if m.SupportsUri(internalUri) {
		return false
	}
	m.withWrite(func() {
		m.trieTree.Add(internalUri, handler, false)
	})
	return true
}

func (m *ServiceHandler) Unregister(internalUri string) bool {
	success := true
	m.withWrite(func() {
		err := m.trieTree.Remove(internalUri)
		if err != nil {
			success = false
		}
	})
	return success
}

func (m *ServiceHandler) GetHandler(internalUri string) RequestHandler {
	m.lock.RLock()
	defer m.lock.RUnlock()
	handler, err := m.trieTree.FindAndGet(internalUri)
	if err != nil {
		return nil
	}
	return handler.(RequestHandler)
}
