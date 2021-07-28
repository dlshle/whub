package http

import (
	"errors"
	"net/http"
	"time"
	"wsdk/common/async"
	"wsdk/common/logger"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/notification"
)

type HTTPWritableConnection struct {
	w      http.ResponseWriter
	addr   string
	logger *logger.SimpleLogger
	b      *async.Barrier
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
	if h.b.IsOpen() {
		h.logger.Println("send to the same HTTP connection more than once")
		return errors.New("unable to send more than once for HTTP connection")
	}
	defer h.b.Open()
	var err error
	h.w.Header().Set("message-id", m.Id())
	h.w.Header().Set("from", m.From())
	h.w.Header().Set("to", m.To())
	// TODO need to add a payload-type as content-type equivalent
	h.w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if m.MessageType() == messages.MessageTypeError {
		h.w.WriteHeader(http.StatusInternalServerError)
		_, err = h.w.Write(m.Payload())
	} else {
		_, err = h.w.Write(m.Payload())
	}
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

func (h *HTTPWritableConnection) Init(w http.ResponseWriter, addr string, logger *logger.SimpleLogger) {
	h.w = w
	h.addr = addr
	h.logger = logger
	h.b = async.NewBarrier()
}

func (h *HTTPWritableConnection) WaitDone() {
	h.b.Wait()
}

func NewHTTPWritableConnection() connection.IConnection {
	return &HTTPWritableConnection{}
}
