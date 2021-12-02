package service

import (
	"errors"
	"fmt"
	"sync"
	"whub/common/connection"
	"whub/hub_common/messages"
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

func NewDefaultServiceHandler() IDefaultServiceHandler {
	return &DefaultServiceHandler{
		uriMap: make(map[string]map[int]RequestHandler),
		lock:   new(sync.RWMutex),
	}
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
	pattern := request.GetContext(ServiceRequestContextUriPattern)
	pathParams := request.GetContext(ServiceRequestContextPathParams)
	queryParams := request.GetContext(ServiceRequestContextQueryParams)
	// TODO, no magic string, but how do we resolve the circular dependency?
	connType := request.GetContext("connection_type")
	isAsyncConnType := connType != nil && connection.IsAsyncType(connType.(uint8))
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
				fmt.Sprintf("can not handle uri %s(%s): unregistered route", request.Uri(), pattern.(string)), request.Headers()))
		}
		handler = requestTypeMap[request.MessageType()]
		if handler == nil {
			if isAsyncConnType && len(requestTypeMap) == 1 {
				// if only 1 requestType and it's async, that's okay to do extra work to assign the right handler for it
				for _, v := range requestTypeMap {
					handler = v
				}
				return
			}
			request.Resolve(messages.NewErrorResponse(request, "",
				messages.MessageTypeSvcMethodNotAllowedError,
				fmt.Sprintf("unsupported method for uri %s", pattern.(string)), request.Headers()))
		}
	})
	if handler != nil {
		return handler(request, pathParams.(map[string]string), queryParams.(map[string]string))
	}
	return nil
}
