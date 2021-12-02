package middleware

import (
	"fmt"
	"whub/common/data_structures"
	"whub/common/logger"
	"whub/hub_common/connection"
	"whub/hub_common/service"
	"whub/hub_server/context"
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
