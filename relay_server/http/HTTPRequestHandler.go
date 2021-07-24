package http

import (
	"net/http"
	"wsdk/common/logger"
	"wsdk/common/utils"
	"wsdk/relay_server/message_dispatcher"
)

type HTTPRequestHandler struct {
	serviceMessageDispatcher *message_dispatcher.ServiceRequestMessageHandler
	logger                   *logger.SimpleLogger
}

type IHTTPRequestHandler interface {
	Handle(r *http.Request) error
}

func (h *HTTPRequestHandler) Handle(w http.ResponseWriter, r *http.Request) (err error) {
	utils.LogError(h.logger, "Handle", err)
	msg, err := TransformRequest(r)
	if err != nil {
		return err
	}
	return h.serviceMessageDispatcher.Handle(msg, NewHTTPWritableConnection(w, r.RemoteAddr))
}
