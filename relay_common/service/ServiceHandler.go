package service

import (
	"sync"
	"wsdk/relay_common/uri"
)

// ServiceHandler handles service requests
type ServiceHandler struct {
	trieTree *uri.TrieTree
	lock     *sync.RWMutex
}

type IServiceHandler interface {
	SupportsUri(uri string) bool
	Register(uri string, handler RequestHandler) bool
	Unregister(uri string) bool
	Handle(request *ServiceRequest) error
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
		success = m.trieTree.Remove(internalUri)
	})
	return success
}

func (m *ServiceHandler) Handle(request *ServiceRequest) error {
	m.lock.RLock()
	defer m.lock.RUnlock()
	matchContext, err := m.trieTree.Match(request.Uri())
	if err != nil {
		// only possible error is no routing found
		return err
	}
	return matchContext.Value.(RequestHandler)(request, matchContext.PathParams, matchContext.QueryParams)
}
