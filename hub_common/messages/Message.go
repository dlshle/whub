package messages

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	common_http "whub/common/http"
	"whub/hub_common/utils"
)

var messagePool *sync.Pool

func init() {
	messagePool = &sync.Pool{
		New: func() interface{} {
			return &Message{}
		},
	}
}

// Message HTTP-Headers
const (
	MessageHTTPHeaderFrom = "X-From"
	MessageHTTPHeaderTo   = "X-To"
	MessageHTTPHeaderId   = "X-Request-Id"
)

// Message Protocol
const (
	MessageProtocolSimple     = 0 // use json string
	MessageProtocolFlatBuffer = 1 // use FlatBuffer
)

// Message Type
const (
	MessageTypeUnknown        = -100
	MessageTypeProtocolUpdate = -1
	MessageTypePing           = 0
	MessageTypePong           = 1
	MessageTypeACK            = 2
	MessageTypeText           = 3
	MessageTypeStream         = 4
	MessageTypeJSON           = 5
	MessageTypeError          = 500

	MessageTypeServiceRequest        = 110
	MessageTypeServiceGetRequest     = 111
	MessageTypeServiceHeadRequest    = 112
	MessageTypeServicePostRequest    = 113
	MessageTypeServicePutRequest     = 114
	MessageTypeServiceDeleteRequest  = 115
	MessageTypeServiceOptionsRequest = 116
	MessageTypeServicePatchRequest   = 117

	MessageTypeServerNotification        = 11
	MessageTypeServerServiceNotification = 12
	MessageTypeClientNotification        = 21
	MessageTypeClientServiceNotification = 22

	MessageTypeServerDescriptor = 100
	MessageTypeClientDescriptor = 200

	MessageTypeInternalNotification = 333

	MessageTypeSvcResponseOK            = 200
	MessageTypeSvcResponseCreated       = 201
	MessageTypeSvcResponsePartial       = 206
	MessageTypeSvcBadRequestError       = 400
	MessageTypeSvcUnauthorizedError     = 401
	MessageTypeSvcForbiddenError        = 403
	MessageTypeSvcNotFoundError         = 404
	MessageTypeSvcMethodNotAllowedError = 405
	MessageTypeSvcGoneError             = 410
	MessageTypeSvcInternalError         = 500
	MessageTypeSvcUnavailableError      = 503
)

type Message struct {
	id          string
	from        string // use id or credential here
	to          string // use id or credential here
	uri         string
	messageType int
	headers     map[string]string
	payload     []byte
}

type IMessage interface {
	Id() string
	From() string
	SetFrom(string) IMessage
	To() string
	SetTo(string) IMessage
	MessageType() int
	SetMessageType(int) IMessage
	Uri() string
	SetUri(string) IMessage
	Payload() []byte
	SetPayload([]byte) IMessage
	String() string
	Copy() IMessage
	GetHeader(key string) string
	SetHeader(key string, value string)
	Headers() map[string]string

	IsErrorMessage() bool
	Dispose()
	ToHTTPRequest(protocol, baseUri, token string) *http.Request
}

func (t *Message) Id() string {
	return t.id
}

func (t *Message) From() string {
	return t.from
}

func (t *Message) SetFrom(from string) IMessage {
	t.from = from
	return t
}

func (t *Message) To() string {
	return t.to
}

func (t *Message) SetTo(to string) IMessage {
	t.to = to
	return t
}

func (t *Message) MessageType() int {
	return t.messageType
}

func (t *Message) SetMessageType(mType int) IMessage {
	t.messageType = mType
	return t
}

func (t *Message) Uri() string {
	return t.uri
}

func (t *Message) SetUri(uri string) IMessage {
	t.uri = uri
	return t
}

func (t *Message) Payload() []byte {
	return t.payload
}

func (t *Message) SetPayload(payload []byte) IMessage {
	t.payload = payload
	return t
}

func (t *Message) GetHeader(key string) string {
	return t.headers[key]
}

func (t *Message) SetHeader(key string, value string) {
	t.headers[key] = value
}

func (t *Message) Headers() map[string]string {
	return t.headers
}

func (t *Message) stringifyHeaders() string {
	var builder strings.Builder
	length := len(t.headers)
	counter := 0
	for k, v := range t.headers {
		counter++
		builder.WriteString(fmt.Sprintf(`"%s": "%s"`, k, v))
		if counter < length {
			builder.WriteByte(',')
		}
	}
	return builder.String()
}

func (t *Message) String() string {
	if t == nil {
		return "nil"
	}
	return fmt.Sprintf("{\"id\":\"%s\",\"from\":\"%s\",\"to\":\"%s\",\"uri\":\"%s\",\"messageType\": %d,\"headers\":{%s},\"payload\":\"%s\"}",
		t.id,
		t.from,
		t.to,
		t.uri,
		t.messageType,
		t.stringifyHeaders(),
		t.payload)
}

func (t *Message) Equals(m IMessage) bool {
	return t.Id() == m.Id() && t.From() == m.From() && t.To() == m.To() && t.Uri() == m.Uri() && t.MessageType() == m.MessageType() && (string)(t.payload) == (string)(m.Payload())
}

func (t *Message) Copy() IMessage {
	newMsg := NewMessage(t.id, t.from, t.to, t.uri, t.messageType, t.payload)
	for k, v := range t.headers {
		newMsg.SetHeader(k, v)
	}
	return newMsg
}

func (t *Message) IsErrorMessage() bool {
	return t.MessageType() == MessageTypeError || t.MessageType() >= MessageTypeSvcBadRequestError && t.MessageType() <= MessageTypeSvcInternalError
}

func (t *Message) Dispose() {
	t.payload = nil
	messagePool.Put(t)
}

func (t *Message) ToHTTPRequest(protocol, baseUri, token string) *http.Request {
	headerMaker := common_http.NewHeaderMaker().
		Set("Id", t.id).
		Set("To", t.To())
	if token != "" {
		headerMaker.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}
	return common_http.NewRequestBuilder().
		Header(headerMaker.Make()).
		URL(fmt.Sprintf("%s://%s%s", protocol, baseUri, t.uri)).
		Method(mapMessageTypeToRequestMethod(t)).
		StringBody((string)(t.payload)).
		Build()
}

func NewMessage(id string, from string, to string, uri string, messageType int, payload []byte) IMessage {
	msg := messagePool.Get().(*Message)
	msg.id = id
	msg.from = from
	msg.to = to
	msg.uri = uri
	msg.messageType = messageType
	msg.payload = payload
	msg.headers = make(map[string]string)
	return msg
}

func DraftMessage(from string, to string, uri string, messageType int, payload []byte) IMessage {
	return NewMessage(utils.GenStringId(), from, to, uri, messageType, payload)
}

func NewInternalErrorMessage(id string, from string, to string, uri string, errorMessage string) IMessage {
	return NewMessage(id, from, to, uri, MessageTypeSvcInternalError, ([]byte)(errorMessage))
}

func NewErrorMessage(id string, from string, to string, uri string, errType int, errorMessage string) IMessage {
	return NewMessage(id, from, to, uri, errType, ([]byte)(errorMessage))
}

func NewPingMessage(from string, to string) IMessage {
	return DraftMessage(from, to, "", MessageTypePing, nil)
}

func NewPongMessage(id string, from string, to string) IMessage {
	return NewMessage(id, from, to, "", MessageTypePong, nil)
}

func NewACKMessage(id string, from string, to string, uri string) IMessage {
	return NewMessage(id, from, to, uri, MessageTypeACK, []byte{})
}

func NewNotification(id string, message string) IMessage {
	return NewMessage(id, "", "", "", MessageTypeInternalNotification, ([]byte)(message))
}

func NewErrorResponse(request IMessage, from string, errType int, errMsg string) IMessage {
	resp := NewMessage(request.Id(), from, request.From(), request.Uri(), errType, ([]byte)(fmt.Sprintf("{\"message\":\"%s\"}", errMsg)))
	return resp
}

func mapMessageTypeToRequestMethod(message *Message) string {
	switch message.messageType {
	case MessageTypeServiceGetRequest:
		return http.MethodGet
	case MessageTypeServicePostRequest:
		return http.MethodPost
	case MessageTypeServicePatchRequest:
		return http.MethodPatch
	case MessageTypeServicePutRequest:
		return http.MethodPut
	case MessageTypeServiceDeleteRequest:
		return http.MethodDelete
	case MessageTypeServiceOptionsRequest:
		return http.MethodOptions
	case MessageTypeServiceHeadRequest:
		return http.MethodHead
	default:
		return http.MethodGet
	}
}
