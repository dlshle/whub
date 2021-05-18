package messages

import (
	"fmt"
)

// Message Protocol
const (
	MessageProtocolSimple     = 0 // use json string
	MessageProtocolFlatBuffer = 1 // use FlatBuffer
)

// Message Type
const (
	MessageTypeProtocolUpdate = -1
	MessageTypePing           = 0
	MessageTypePong           = 1
	MessageTypeACK            = 2
	MessageTypeText           = 3
	MessageTypeStream         = 4
	MessageTypeJSON           = 5
	MessageTypeError          = 6

	MessageTypeServerNotification = 11
	MessageTypeServerServiceNotification = 12
	MessageTypeClientNotification = 21
	MessageTypeClientServiceNotification = 22

	MessageTypeServerDescriptor = 100
	MessageTypeClientDescriptor = 200
)

type Message struct {
	id          string
	from        string // use id or credential here
	to          string // use id or credential here
	uri			string
	messageType int
	payload     []byte
}

type IMessage interface {
	Id() string
	From() string
	To() string
	MessageType() int
	Uri() string
	Payload() []byte
	String() string
}

func (t *Message) Id() string {
	return t.id
}

func (t *Message) From() string {
	return t.from
}

func (t *Message) To() string {
	return t.to
}

func (t *Message) MessageType() int {
	return t.messageType
}

func (t *Message) Uri() string {
	return t.uri
}

func (t *Message) Payload() []byte {
	return t.payload
}

func (t *Message) String() string {
	return fmt.Sprintf("{from: \"%s\", to: \"%s\", messageType: %d, payload: %s}", t.from, t.to, t.messageType, t.payload)
}


func NewMessage(id string, from string, to string, uri string, messageType int, payload []byte) *Message {
	return &Message{id, from, to, uri, messageType, payload}
}

func NewErrorMessage(id string, from string, to string, uri string, errorMessage string) *Message {
	return &Message{id, from, to, uri, MessageTypeError, ([]byte)(errorMessage)}
}

func NewPingMessage(id string, from string, to string) *Message {
	return &Message{id, from, to, "", MessageTypePing, nil}
}

func NewPongMessage(id string, from string, to string) *Message {
	return &Message{id, from, to, "", MessageTypePong, nil}
}

