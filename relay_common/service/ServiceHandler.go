package service

import (
	"errors"
	"fmt"
	"sync"
	uri2 "wsdk/common/uri_trie"
)

// ServiceHandler handles service_manager requests
type ServiceHandler struct {
	trieTree *uri2.TrieTree
	lock     *sync.RWMutex
}

type IServiceHandler interface {
	SupportsUri(uri string) bool
	Register(uri string, handler RequestHandler) error
	Unregister(uri string) error
	Handle(request *ServiceRequest) error
}

func NewServiceHandler() IServiceHandler {
	return &ServiceHandler{
		trieTree: uri2.NewTrieTree(),
		lock:     new(sync.RWMutex),
	}
}

func (m *ServiceHandler) withWrite(cb func()) {
	m.lock.Lock()
	defer m.lock.Unlock()
	cb()
}

func (m *ServiceHandler) SupportsUri(uri string) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.trieTree.SupportsUri(uri)
}

func (m *ServiceHandler) Register(uri string, handler RequestHandler) (err error) {
	if m.SupportsUri(uri) {
		return errors.New(fmt.Sprintf("uri_trie %s has already been registered", uri))
	}
	m.withWrite(func() {
		err = m.trieTree.Add(uri, handler, false)
	})
	return err
}

func (m *ServiceHandler) Unregister(uri string) (err error) {
	success := true
	m.withWrite(func() {
		success = m.trieTree.Remove(uri)
	})
	if !success {
		err = errors.New(fmt.Sprintf("unable to remove uri_trie %s as it's not registered into service_manager handler", uri))
	}
	return err
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
