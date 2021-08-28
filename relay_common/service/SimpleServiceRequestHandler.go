package service

import (
	"errors"
	"fmt"
	"sync"
)

// SimpleRequestHandler handles internal service requests
// made this to reduce the usage of trie_tree for each internal services, too expensive
type SimpleRequestHandler struct {
	uriMap map[string]RequestHandler
	lock   *sync.RWMutex
}

type ISimpleRequestHandler interface {
	Register(uri string, handler RequestHandler) error
	Unregister(uri string) error
	Handle(request IServiceRequest) error
}

func NewSimpleServiceHandler() ISimpleRequestHandler {
	return &SimpleRequestHandler{
		uriMap: make(map[string]RequestHandler),
		lock:   new(sync.RWMutex),
	}
}

func (m *SimpleRequestHandler) withWrite(cb func()) {
	m.lock.Lock()
	defer m.lock.Unlock()
	cb()
}

func (m *SimpleRequestHandler) Register(uri string, handler RequestHandler) (err error) {
	if m.uriMap[uri] != nil {
		return errors.New(fmt.Sprintf("uri_trie %s has already been registered", uri))
	}
	m.withWrite(func() {
		m.uriMap[uri] = handler
	})
	return nil
}

func (m *SimpleRequestHandler) Unregister(uri string) (err error) {
	if m.uriMap[uri] == nil {
		return errors.New(fmt.Sprintf("unable to remove uri_trie %s as it's not registered into service handler", uri))
	}
	m.withWrite(func() {
		delete(m.uriMap, uri)
	})
	return nil
}

func (m *SimpleRequestHandler) Handle(request IServiceRequest) error {
	m.lock.RLock()
	defer m.lock.RUnlock()
	// TODO this is bad, we need less uncertainty and need to add UriPattern, PathParams, QueryParams to IServiceRequest
	pattern := request.GetContext("uri_pattern")
	pathParams := request.GetContext("path_params")
	queryParams := request.GetContext("query_params")
	if pattern == nil {
		return errors.New("unable to get uri pattern from request")
	}
	if pathParams == nil || queryParams == nil {
		return errors.New("unable to get params from request")
	}
	handler := m.uriMap[pattern.(string)]
	if handler == nil {
		return errors.New(fmt.Sprintf("unable to process uri %s", pattern.(string)))
	}
	return handler(request, pathParams.(map[string]string), queryParams.(map[string]string))
}
