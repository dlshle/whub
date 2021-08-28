package service

import (
	"errors"
	"fmt"
	"sync"
	"wsdk/relay_common/messages"
)

type IDefaultServiceHandler interface {
	Register(requestType int, uri string, handler RequestHandler) error
	Unregister(requestType int, uri string) error
	Handle(request IServiceRequest) error
}

type DefaultServiceHandler struct {
	uriMap map[string]map[int]RequestHandler
	lock   *sync.RWMutex
}

func (h *DefaultServiceHandler) withWrite(cb func()) {
	h.lock.Lock()
	defer h.lock.Unlock()
	cb()
}

func (h *DefaultServiceHandler) withRead(cb func()) {
	h.lock.RLock()
	defer h.lock.RUnlock()
	cb()
}

func (h *DefaultServiceHandler) Register(requestType int, uri string, handler RequestHandler) error {
	if requestType < 100 || requestType > 200 {
		return errors.New("invalid request type")
	}
	h.withWrite(func() {
		if h.uriMap[uri] == nil {
			h.uriMap[uri] = make(map[int]RequestHandler)
		}
		h.uriMap[uri][requestType] = handler
	})
	return nil
}

func (h *DefaultServiceHandler) Unregister(requestType int, uri string) (err error) {
	if requestType < 100 || requestType > 200 {
		return errors.New("invalid request type")
	}
	h.withRead(func() {
		if h.uriMap[uri] == nil {
			err = errors.New(fmt.Sprintf("uri %s is not registered into the service handler", uri))
		}
	})
	if err != nil {
		return
	}
	h.withWrite(func() {
		delete(h.uriMap[uri], requestType)
		if len(h.uriMap[uri]) == 0 {
			delete(h.uriMap, uri)
		}
	})
	return nil
}

func (h *DefaultServiceHandler) Handle(request IServiceRequest) (err error) {
	var handler RequestHandler
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
	h.withRead(func() {
		requestTypeMap := h.uriMap[pattern.(string)]
		if requestTypeMap == nil {
			request.Resolve(messages.NewErrorResponse(request, "",
				messages.MessageTypeSvcNotFoundError,
				fmt.Sprintf("can not handle uri %s(%s): unregistered route", request.Uri(), pattern.(string))))
		}
		handler = requestTypeMap[request.MessageType()]
		if handler == nil {
			request.Resolve(messages.NewErrorResponse(request, "",
				messages.MessageTypeSvcMethodNotAllowedError,
				fmt.Sprintf("unsupported method for uri %s", pattern.(string))))
		}
	})
	if handler != nil {
		return handler(request, pathParams.(map[string]string), queryParams.(map[string]string))
	}
	return nil
}
