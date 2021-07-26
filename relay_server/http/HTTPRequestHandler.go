package http

import (
	"fmt"
	"net/http"
	"wsdk/common/logger"
	"wsdk/common/utils"
	"wsdk/relay_server/context"
	"wsdk/relay_server/message_dispatcher"
)

type HTTPRequestHandler struct {
	serviceMessageDispatcher *message_dispatcher.ServerMessageDispatcher
	logger                   *logger.SimpleLogger
}

func NewHTTPRequestHandler(dispatcher *message_dispatcher.ServerMessageDispatcher) IHTTPRequestHandler {
	return &HTTPRequestHandler{
		dispatcher,
		context.Ctx.Logger().WithPrefix("[HTTPRequestHandler]"),
	}
}

type IHTTPRequestHandler interface {
	Handle(w http.ResponseWriter, r *http.Request)
}

func (h *HTTPRequestHandler) Handle(w http.ResponseWriter, r *http.Request) {
	h.logger.Println("handle incoming HTTP request: ", r.RequestURI, r.Header)
	msg, err := TransformRequest(r)
	if err != nil {
		utils.LogError(h.logger, "Handle", err)
	}
	h.serviceMessageDispatcher.Dispatch(msg, NewHTTPWritableConnection(w, r.RemoteAddr, h.logger.WithPrefix(fmt.Sprintf("[HTTP-Conn-%s]", r.RemoteAddr))))
}
