package http

import (
	"fmt"
	"net/http"
	"wsdk/common/logger"
	"wsdk/common/utils"
	"wsdk/relay_common/message_actions"
	"wsdk/relay_server/context"
)

type HTTPRequestHandler struct {
	serviceMessageDispatcher message_actions.IMessageHandler
	logger                   *logger.SimpleLogger
}

func NewHTTPRequestHandler(dispatcher message_actions.IMessageHandler) IHTTPRequestHandler {
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
	// Do not do this on another goroutine. It will cause issue with ResponseWriter.
	h.serviceMessageDispatcher.Handle(msg, NewHTTPWritableConnection(w, r.RemoteAddr, h.logger.WithPrefix(fmt.Sprintf("[HTTP-%s-%s]", r.RemoteAddr, msg.Id()))))
}
