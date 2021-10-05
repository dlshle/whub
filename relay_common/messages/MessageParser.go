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
	// payload
	payload := message.Payload()
	lPayload := len(payload)
	Flatbuffer_Message.MessageStartPayloadVector(builder, lPayload)
	for i := range payload {
		builder.PrependByte(payload[lPayload-i-1])
	}
	payloadOffset := builder.EndVector(lPayload)
	// headers
	headers := message.Headers()
	lHeaders := len(headers)
	Flatbuffer_Message.MessageStartHeaderKeysVector(builder, lHeaders)
	for k, _ := range headers {
		builder.CreateString(k)
	}
	headerKeysOffset := builder.EndVector(lHeaders)
	Flatbuffer_Message.MessageStartHeaderValuesVector(builder, lHeaders)
	for _, v := range headers {
		builder.CreateString(v)
	}
	headerValuesOffset := builder.EndVector(lHeaders)

	// primitives
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
	Flatbuffer_Message.MessageAddHeaderKeys(builder, headerKeysOffset)
	Flatbuffer_Message.MessageAddHeaderValues(builder, headerValuesOffset)
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
	// headers
	message := NewMessage(id, from, to, uri, (int)(msgType), payload)
	headerLen := fbMessage.HeaderKeysLength()
	if headerLen != fbMessage.HeaderValuesLength() {
		return nil, errors.New("invalid message format")
	}
	for i := 0; i < headerLen; i++ {
		message.SetHeader((string)(fbMessage.HeaderKeys(i)), (string)(fbMessage.HeaderValues(i)))
	}
	return message, nil
}
