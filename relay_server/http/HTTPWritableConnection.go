package http

import (
	"net/http"
	"time"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/notification"
)

type HTTPWritableConnection struct {
	w    http.ResponseWriter
	addr string
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
	h.w.Header().Set("message-id", m.Id())
	h.w.Header().Set("from", m.From())
	h.w.Header().Set("to", m.To())
	if m.MessageType() == messages.MessageTypeError {
		h.w.WriteHeader(500)
	}
	_, err := h.w.Write(m.Payload())
	return err
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

func NewHTTPWritableConnection(w http.ResponseWriter, addr string) connection.IConnection {
	return &HTTPWritableConnection{
		w,
		addr,
	}
}
