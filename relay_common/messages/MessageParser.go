package messages

import (
	"errors"
	flatbuffers "github.com/google/flatbuffers/go"
	Flatbuffer_Message "wsdk/relay_common/flatbuffers/WR/Message"
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
	Flatbuffer_Message.MessageStartPayloadVector(builder, lPayload)
	for i := range payload {
		builder.PrependByte(payload[lPayload-i-1])
	}
	payloadOffset := builder.EndVector(lPayload)
	idOffset := builder.CreateString(message.Id())
	fromOffset := builder.CreateString(message.From())
	toOffset := builder.CreateString(message.To())
	uriOffset := builder.CreateString(message.Uri())
	Flatbuffer_Message.MessageStart(builder)
	Flatbuffer_Message.MessageAddId(builder, idOffset)
	Flatbuffer_Message.MessageAddFrom(builder, fromOffset)
	Flatbuffer_Message.MessageAddTo(builder, toOffset)
	Flatbuffer_Message.MessageAddUri(builder, uriOffset)
	Flatbuffer_Message.MessageAddMessageType(builder, (int32)(message.MessageType()))
	Flatbuffer_Message.MessageAddPayload(builder, payloadOffset)
	offset := Flatbuffer_Message.MessageEnd(builder)
	builder.Finish(offset)
	return builder.Bytes[builder.Head():], nil
}

func (p *FBMessageParser) Deserialize(buffer []byte) (msg IMessage, err error) {
	defer func() {
		panicMsg := recover()
		if panicMsg != nil {
			err = errors.New("unable to parse the message")
		}
	}()
	if len(buffer) < 1 {
		return nil, errors.New("invalid buffer format")
	}
	fbMessage := Flatbuffer_Message.GetRootAsMessage(buffer, 0)
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
