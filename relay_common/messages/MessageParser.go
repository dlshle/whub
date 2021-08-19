package messages

import (
	"errors"
	flatbuffers "github.com/google/flatbuffers/go"
	Message2 "wsdk/relay_common/flatbuffers/WR/Message"
)

type IMessageParser interface {
	Serialize(message IMessage) ([]byte, error)
	Deserialize([]byte) (IMessage, error)
}

type FBMessageParser struct{}

func NewFBMessageParser() *FBMessageParser {
	return &FBMessageParser{}
}

func (p *FBMessageParser) Serialize(message IMessage) ([]byte, error) {
	builder := flatbuffers.NewBuilder(16)
	payload := message.Payload()
	lPayload := len(payload)
	Message2.MessageStartPayloadVector(builder, lPayload)
	for i := range payload {
		builder.PrependByte(payload[lPayload-i-1])
	}
	payloadOffset := builder.EndVector(lPayload)
	idOffset := builder.CreateString(message.Id())
	fromOffset := builder.CreateString(message.From())
	toOffset := builder.CreateString(message.To())
	uriOffset := builder.CreateString(message.Uri())
	Message2.MessageStart(builder)
	Message2.MessageAddId(builder, idOffset)
	Message2.MessageAddFrom(builder, fromOffset)
	Message2.MessageAddTo(builder, toOffset)
	Message2.MessageAddUri(builder, uriOffset)
	Message2.MessageAddMessageType(builder, (int32)(message.MessageType()))
	Message2.MessageAddPayload(builder, payloadOffset)
	offset := Message2.MessageEnd(builder)
	builder.Finish(offset)
	return builder.Bytes[builder.Head():], nil
}

func (p *FBMessageParser) Deserialize(buffer []byte) (IMessage, error) {
	if len(buffer) < 1 {
		return nil, errors.New("invalid buffer format")
	}
	fbMessage := Message2.GetRootAsMessage(buffer, 0)
	id := (string)(fbMessage.Id())
	from := (string)(fbMessage.From())
	to := (string)(fbMessage.To())
	uri := (string)(fbMessage.Uri())
	msgType := fbMessage.MessageType()
	payload := fbMessage.Payload()
	if err := recover(); err != nil {
		return nil, errors.New("unable to deserialize message")
	}
	return NewMessage(id, from, to, uri, (int)(msgType), payload), nil
}
