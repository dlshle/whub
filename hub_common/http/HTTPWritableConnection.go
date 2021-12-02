package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
	"whub/common/async"
	base_conn "whub/common/connection"
	c_http "whub/common/http"
	"whub/common/logger"
	"whub/hub_common/connection"
	"whub/hub_common/messages"
	"whub/hub_common/notification"
)

type HTTPWritableConnection struct {
	w        http.ResponseWriter
	addr     string
	logger   *logger.SimpleLogger
	waitLock *async.WaitLock
	isWhr    bool
}

func (h *HTTPWritableConnection) Address() string {
	return h.addr
}

func (h *HTTPWritableConnection) ReadingLoop() {
	return
}

func (h *HTTPWritableConnection) Request(message messages.IMessage) (messages.IMessage, error) {
	err := h.Send(message)
	return nil, err
}

func (h *HTTPWritableConnection) RequestWithTimeout(message messages.IMessage, duration time.Duration) (messages.IMessage, error) {
	return h.Request(message)
}

func (h *HTTPWritableConnection) Send(m messages.IMessage) error {
	if h.waitLock.IsOpen() {
		h.logger.Println("send to the same HTTP connection more than once")
		return errors.New("unable to send more than once for HTTP connection")
	}
	defer h.waitLock.Open()
	defer m.Dispose()
	var err error
	if h.isWhr {
		err = h.writeWhrResponse(m)
	} else {
		err = h.writeMessageResponse(m)
	}
	if err != nil {
		h.logger.Println("response write error: ", err.Error())
		return err
	}
	return nil
}

func (h *HTTPWritableConnection) writeMessageResponse(m messages.IMessage) (err error) {
	h.w.Header().Set(messages.MessageHTTPHeaderId, m.Id())
	h.w.Header().Set(messages.MessageHTTPHeaderFrom, m.From())
	h.w.Header().Set(messages.MessageHTTPHeaderTo, m.To())

	// TODO need to add a payload-type as content-type equivalent
	if m.GetHeader("Content-Type") == "" {
		h.w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	}
	h.w.WriteHeader(m.MessageType())
	h.writeMessageHeaders(m)
	_, err = h.w.Write(m.Payload())
	return
}

func (h *HTTPWritableConnection) writeMessageHeaders(m messages.IMessage) {
	for k, v := range m.Headers() {
		h.w.Header().Set(k, v)
	}
}

func (h *HTTPWritableConnection) writeWhrResponse(m messages.IMessage) (err error) {
	payload := m.Payload()
	var response c_http.Response
	err = json.Unmarshal(payload, &response)
	if err != nil || response.Code == 0 {
		// fallback strategy
		return h.writeMessageResponse(m)
	}
	if response.Code < 0 {
		response.Code = http.StatusInternalServerError
	}
	h.w.WriteHeader(response.Code)
	for k, v := range response.Header {
		h.w.Header().Set(k, v[0])
	}
	_, err = h.w.Write(([]byte)(response.Body))
	return
}

func (h *HTTPWritableConnection) OnIncomingMessage(f func(message messages.IMessage)) {
}

func (h *HTTPWritableConnection) OnceMessage(s string, f func(messages.IMessage)) (notification.Disposable, error) {
	return nil, nil
}

func (h *HTTPWritableConnection) OnMessage(s string, f func(messages.IMessage)) (notification.Disposable, error) {
	return nil, nil
}

func (h *HTTPWritableConnection) OffMessage(s string, f func(messages.IMessage)) {
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

func (h *HTTPWritableConnection) Init(w http.ResponseWriter, addr string, logger *logger.SimpleLogger, isWhr bool) {
	h.w = w
	h.addr = addr
	h.logger = logger
	h.waitLock = async.NewWaitLock()
	h.isWhr = isWhr
}

func (h *HTTPWritableConnection) WaitDone() {
	h.waitLock.Wait()
}

func (h *HTTPWritableConnection) ConnectionType() uint8 {
	return base_conn.TypeHTTP
}

func (h *HTTPWritableConnection) IsLive() bool {
	return h.waitLock.IsOpen()
}

func NewHTTPWritableConnection() connection.IConnection {
	return &HTTPWritableConnection{}
}
