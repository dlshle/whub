package http

import (
	"errors"
	"fmt"
	"net/http"
	"time"
	"wsdk/common/async"
	base_conn "wsdk/common/connection"
	"wsdk/common/logger"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/notification"
)

type HTTPWritableConnection struct {
	w        http.ResponseWriter
	addr     string
	logger   *logger.SimpleLogger
	waitLock *async.WaitLock
}

func (h *HTTPWritableConnection) Address() string {
	return h.addr
}

func (h *HTTPWritableConnection) ReadingLoop() {
	return
}

func (h *HTTPWritableConnection) Request(message *messages.Message) (*messages.Message, error) {
	err := h.Send(message)
	return nil, err
}

func (h *HTTPWritableConnection) RequestWithTimeout(message *messages.Message, duration time.Duration) (*messages.Message, error) {
	return h.Request(message)
}

func (h *HTTPWritableConnection) Send(m *messages.Message) error {
	if h.waitLock.IsOpen() {
		h.logger.Println("send to the same HTTP connection more than once")
		return errors.New("unable to send more than once for HTTP connection")
	}
	defer h.waitLock.Open()
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
}

func (h *HTTPWritableConnection) OnceMessage(s string, f func(*messages.Message)) (notification.Disposable, error) {
	return nil, nil
}

func (h *HTTPWritableConnection) OnMessage(s string, f func(*messages.Message)) (notification.Disposable, error) {
	return nil, nil
}

func (h *HTTPWritableConnection) OffMessage(s string, f func(*messages.Message)) {
}

func (h *HTTPWritableConnection) OffAll(s string) {
}

func (h *HTTPWritableConnection) OnError(f func(error)) {
}

func (h *HTTPWritableConnection) OnClose(f func(error)) {
}

func (h *HTTPWritableConnection) Close() error {
	return h.Send(messages.NewACKMessage("", "", h.addr, h.addr))
}

func (h *HTTPWritableConnection) String() string {
	return fmt.Sprintf("{\"type\":\"%s\",\"address\":\"%s\"}", base_conn.TypeString(base_conn.TypeHTTP), h.Address())
}

func (h *HTTPWritableConnection) Init(w http.ResponseWriter, addr string, logger *logger.SimpleLogger) {
	h.w = w
	h.addr = addr
	h.logger = logger
	h.waitLock = async.NewWaitLock()
}

func (h *HTTPWritableConnection) WaitDone() {
	h.waitLock.Wait()
}

func (h *HTTPWritableConnection) ConnectionType() uint8 {
	return base_conn.TypeHTTP
}

func NewHTTPWritableConnection() connection.IConnection {
	return &HTTPWritableConnection{}
}
