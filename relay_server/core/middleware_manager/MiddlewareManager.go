package middleware_manager

import (
	"wsdk/relay_common/connection"
	"wsdk/relay_common/service"
	"wsdk/relay_server/container"
	"wsdk/relay_server/middleware"
)

const MaxMiddlewareCount = 64

type IMiddlewareManager interface {
	RunMiddlewares(conn connection.IConnection, request service.IServiceRequest) service.IServiceRequest
	AddMiddleware(middleware middleware.RequestMiddleware)
	Clear()
}

type MiddlewareManager struct {
	middlewares []middleware.RequestMiddleware
}

func NewMiddlewareManager() IMiddlewareManager {
	return &MiddlewareManager{
		middlewares: make([]middleware.RequestMiddleware, 0, MaxMiddlewareCount),
	}
}

func (m *MiddlewareManager) RunMiddlewares(conn connection.IConnection, request service.IServiceRequest) service.IServiceRequest {
	for _, middleware := range m.middlewares {
		request = middleware(conn, request)
	}
	return request
}

func (m *MiddlewareManager) AddMiddleware(requestMiddleware middleware.RequestMiddleware) {
	if len(m.middlewares) >= MaxMiddlewareCount {
		return
	}
	m.middlewares = append(m.middlewares, requestMiddleware)
}

func (m *MiddlewareManager) Clear() {
	m.middlewares = nil
	m.middlewares = make([]middleware.RequestMiddleware, 0, MaxMiddlewareCount)
}

func init() {
	container.Container.Singleton(func() IMiddlewareManager {
		return NewMiddlewareManager()
	})
}
