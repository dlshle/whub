package middleware

import (
	"fmt"
	"wsdk/common/data_structures"
	"wsdk/common/logger"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/service"
	"wsdk/relay_server/context"
)

type IServerMiddleware interface {
	Init() error
	Id() string
	Run(conn connection.IConnection, request service.IServiceRequest) service.IServiceRequest
	Logger() *logger.SimpleLogger
	Priority() int
	Compare(comparable data_structures.IComparable) int
}

type ServerMiddleware struct {
	id       string
	logger   *logger.SimpleLogger
	priority int
}

func NewServerMiddleware(id string, priority int) *ServerMiddleware {
	return &ServerMiddleware{
		id:       id,
		priority: priority,
		logger:   context.Ctx.Logger().WithPrefix(fmt.Sprintf("[Middleware-%s]", id)),
	}
}

func (m *ServerMiddleware) Id() string {
	return m.id
}

func (m *ServerMiddleware) Logger() *logger.SimpleLogger {
	return m.logger
}

func (m *ServerMiddleware) Compare(a data_structures.IComparable) int {
	t := a.(IServerMiddleware)
	return m.priority - t.Priority()
}

func (m *ServerMiddleware) Priority() int {
	return m.priority
}
