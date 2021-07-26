package http

import (
	"fmt"
	"net/http"
	"time"
	"wsdk/common/logger"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/notification"
)

type HTTPWritableConnection struct {
	w      http.ResponseWriter
	addr   string
	logger *logger.SimpleLogger
}

func (h *HTTPWritableConnection) Address() string {
	return h.addr
}

func (h *HTTPWritableConnection) StartListening() {
	panic("implement me")
}

func (h *HTTPWritableConnection) ReadingLoop() {
	panic("implement me")
}

func (h *HTTPWritableConnection) Request(message *messages.Message) (*messages.Message, error) {
	panic("implement me")
}

func (h *HTTPWritableConnection) RequestWithTimeout(message *messages.Message, duration time.Duration) (*messages.Message, error) {
	panic("implement me")
}

func (h *HTTPWritableConnection) Send(m *messages.Message) error {
	h.logger.Println("response message: ", m)
	h.w.Header().Set("message-id", m.Id())
	h.w.Header().Set("from", m.From())
	h.w.Header().Set("to", m.To())
	// TODO some error here
	if m.MessageType() == messages.MessageTypeError {
		http.Error(h.w, (string)(m.Payload()), http.StatusInternalServerError)
		return nil
	}
	_, err := fmt.Fprintf(h.w, "%s", (string)(m.Payload()))
	if err != nil {
		h.logger.Println("response write error: ", err.Error())
		return err
	}
	return nil
}

func (h *HTTPWritableConnection) OnIncomingMessage(f func(message *messages.Message)) {
	panic("implement me")
}

func (h *HTTPWritableConnection) OnceMessage(s string, f func(*messages.Message)) (notification.Disposable, error) {
	panic("implement me")
}

func (h *HTTPWritableConnection) OnMessage(s string, f func(*messages.Message)) (notification.Disposable, error) {
	panic("implement me")
}

func (h *HTTPWritableConnection) OffMessage(s string, f func(*messages.Message)) {
	panic("implement me")
}

func (h *HTTPWritableConnection) OffAll(s string) {
	panic("implement me")
}

func (h *HTTPWritableConnection) OnError(f func(error)) {
	panic("implement me")
}

func (h *HTTPWritableConnection) OnClose(f func(error)) {
	panic("implement me")
}

func (h *HTTPWritableConnection) Close() error {
	panic("implement me")
}

func NewHTTPWritableConnection(w http.ResponseWriter, addr string, logger *logger.SimpleLogger) connection.IConnection {
	return &HTTPWritableConnection{
		w,
		addr,
		logger,
	}
}
