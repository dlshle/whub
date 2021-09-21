package http

import (
	"fmt"
	"net/http"
	"sync"
	"wsdk/common/logger"
	"wsdk/relay_common/dispatcher"
	whttp "wsdk/relay_common/http"
	"wsdk/relay_server/context"
)

type HTTPRequestHandler struct {
	serviceMessageDispatcher dispatcher.IMessageDispatcher
	logger                   *logger.SimpleLogger
	pool                     *sync.Pool
}

func NewHTTPRequestHandler(dispatcher dispatcher.IMessageDispatcher) IHTTPRequestHandler {
	pool := &sync.Pool{New: func() interface{} {
		return whttp.NewHTTPWritableConnection()
	}}
	return &HTTPRequestHandler{
		dispatcher,
		context.Ctx.Logger().WithPrefix("[HTTPRequestHandler]"),
		pool,
	}
}

type IHTTPRequestHandler interface {
	Handle(w http.ResponseWriter, r *http.Request)
}

func (h *HTTPRequestHandler) Handle(w http.ResponseWriter, r *http.Request) {
	h.logger.Println("handle incoming HTTP request: ", r.RequestURI, r.Header)
	msg, err := TransformRequest(r)
	if err != nil {
		logger.LogError(h.logger, "Handle", err)
	}
	conn := h.pool.Get().(*whttp.HTTPWritableConnection)
	conn.Init(w, r.RemoteAddr, h.logger.WithPrefix(fmt.Sprintf("[HTTP-%s-%s]", r.RemoteAddr, msg.Id())), isWhrRequest(r))
	// Do not do this on another goroutine. It will cause issue with ResponseWriter.
	h.serviceMessageDispatcher.Dispatch(msg, conn)
	conn.WaitDone()
	// recycle after conn is used
	h.pool.Put(conn)
}
