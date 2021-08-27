package request

import (
	"errors"
	"fmt"
	"sync"
	"wsdk/relay_common/service"
)

// InternalServiceHandler handles internal service requests
// made this to reduce the usage of trie_tree for each internal services, too expensive
type InternalServiceHandler struct {
	uriMap map[string]service.RequestHandler
	lock   *sync.RWMutex
}

type IInternalServiceHandler interface {
	Register(uri string, handler service.RequestHandler) error
	Unregister(uri string) error
	Handle(request service.IServiceRequest) error
}

func NewServiceHandler() IInternalServiceHandler {
	return &InternalServiceHandler{
		uriMap: make(map[string]service.RequestHandler),
		lock:   new(sync.RWMutex),
	}
}

func (m *InternalServiceHandler) withWrite(cb func()) {
	m.lock.Lock()
	defer m.lock.Unlock()
	cb()
}

func (m *InternalServiceHandler) Register(uri string, handler service.RequestHandler) (err error) {
	if m.uriMap[uri] != nil {
		return errors.New(fmt.Sprintf("uri_trie %s has already been registered", uri))
	}
	m.withWrite(func() {
		m.uriMap[uri] = handler
	})
	return nil
}

func (m *InternalServiceHandler) Unregister(uri string) (err error) {
	if m.uriMap[uri] == nil {
		return errors.New(fmt.Sprintf("unable to remove uri_trie %s as it's not registered into service handler", uri))
	}
	m.withWrite(func() {
		delete(m.uriMap, uri)
	})
	return nil
}

func (m *InternalServiceHandler) Handle(request service.IServiceRequest) error {
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
