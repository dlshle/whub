package messages

import (
	"fmt"
	"wsdk/relay_common/utils"
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
	MessageTypeError          = 6

	MessageTypeServiceRequest  = 101
	MessageTypeServiceResponse = 102

	MessageTypeServerNotification        = 11
	MessageTypeServerServiceNotification = 12
	MessageTypeClientNotification        = 21
	MessageTypeClientServiceNotification = 22

	MessageTypeServerDescriptor = 100
	MessageTypeClientDescriptor = 200

	MessageTypeInternalNotification = 333

	MessageTypeSvcResponseOK        = 200
	MessageTypeSvcResponseCreated   = 201
	MessageTypeSvcResponsePartial   = 206
	MessageTypeSvcBadRequestError   = 400
	MessageTypeSvcUnauthorizedError = 401
	MessageTypeSvcForbiddenError    = 403
	MessageTypeSvcNotFoundError     = 404
	MessageTypeSvcGoneError         = 410
	MessageTypeSvcInternalError     = 500
)

type Message struct {
	id          string
	from        string // use id or credential here
	to          string // use id or credential here
	uri         string
	messageType int
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

func (t *Message) String() string {
	if t == nil {
		return "nil"
	}
	return fmt.Sprintf("{id: \"%s\", from: \"%s\", to: \"%s\", uri: \"%s\", messageType: %d, payload: \"%s\"}", t.id, t.from, t.to, t.uri, t.messageType, t.payload)
}

func (t *Message) Equals(m IMessage) bool {
	return t.Id() == m.Id() && t.From() == m.From() && t.To() == m.To() && t.Uri() == m.Uri() && t.MessageType() == m.MessageType() && (string)(t.payload) == (string)(m.Payload())
}

func (t *Message) Copy() IMessage {
	return NewMessage(t.id, t.from, t.to, t.uri, t.messageType, t.payload)
}

func NewMessage(id string, from string, to string, uri string, messageType int, payload []byte) IMessage {
	return &Message{id, from, to, uri, messageType, payload}
}

func DraftMessage(from string, to string, uri string, messageType int, payload []byte) IMessage {
	return NewMessage(utils.GenStringId(), from, to, uri, messageType, payload)
}

func NewErrorMessage(id string, from string, to string, uri string, errorMessage string) IMessage {
	return &Message{id, from, to, uri, MessageTypeError, ([]byte)(errorMessage)}
}

func NewPingMessage(id string, from string, to string) IMessage {
	return &Message{id, from, to, "", MessageTypePing, nil}
}

func NewPongMessage(id string, from string, to string) IMessage {
	return &Message{id, from, to, "", MessageTypePong, nil}
}

func NewACKMessage(id string, from string, to string, uri string) IMessage {
	return &Message{id: id, from: from, to: to, uri: uri, messageType: MessageTypeACK}
}

func NewNotification(id string, message string) IMessage {
	return &Message{id: id, messageType: MessageTypeInternalNotification, payload: ([]byte)(message)}
}

func NewErrorResponseMessage(request IMessage, from string, errMsg string) IMessage {
	return &Message{id: request.Id(), from: from, to: request.From(), uri: request.Uri(), messageType: MessageTypeError, payload: ([]byte)(errMsg)}
}

func IsErrorMessage(message IMessage) bool {
	return message.MessageType() >= MessageTypeSvcBadRequestError && message.MessageType() <= MessageTypeSvcInternalError
}
