package messages

import (
	"errors"
	"fmt"
	flatbuffers "github.com/google/flatbuffers/go"
	"strconv"
	FBMessage "wsdk/flatbuffers/WR/Message"
)

type IMessageParser interface {
	Serialize(message *Message) ([]byte, error)
	Deserialize([]byte) (*Message, error)
}

type SimpleMessageParser struct{}

func NewSimpleMessageParser() *SimpleMessageParser {
	return &SimpleMessageParser{}
}

func (h *SimpleMessageParser) Serialize(message *Message) ([]byte, error) {
	return ([]byte)(fmt.Sprintf("%s*%s*%s*%s*%d*%s", message.Id(), message.From(), message.To(), message.Uri(), message.MessageType(), message.Payload())), nil
}

func (h *SimpleMessageParser) Deserialize(serialMessage []byte) (msg *Message, err error) {
	last := 0
	stage := 0
	size := len(serialMessage)
	lastIndex := 4
	var id, msgFrom, msgTo, msgUri string
	var msgType int
	var payload []byte
	hasError := false
	stageMap := make(map[int]func(int, int))
	stageMap[0] = func(from, to int) {
		id = (string)(serialMessage[0 : to+1])
	}
	stageMap[1] = func(from, to int) {
		msgFrom = (string)(serialMessage[from : to+1])
	}
	stageMap[2] = func(from, to int) {
		msgTo = (string)(serialMessage[from : to+1])
	}
	stageMap[3] = func(from, to int) {
		msgUri = (string)(serialMessage[from : to+1])
	}
	stageMap[4] = func(from, to int) {
		msgType, err = strconv.Atoi((string)(serialMessage[from : to+1]))
		if err != nil {
			hasError = true
		}
	}
	stageMap[5] = func(from, to int) {
		payload = serialMessage[from:size]
	}
	for i, c := range serialMessage {
		if c == '*' {
			stageMap[stage](last, i)
			if hasError {
				return nil, err
			}
			last = i + 1
			stage++
			if stage == lastIndex {
				// i == index of the last *
				stageMap[stage](i, -1)
				break
			}
		}
	}
	if stage != lastIndex {
		return nil, errors.New("invalid messages format")
	}
	return NewMessage(id, msgFrom, msgTo, msgUri, msgType, payload), nil
}

type FBMessageParser struct{}

func NewFBMessageParser() *FBMessageParser {
	return &FBMessageParser{}
}

func (p *FBMessageParser) Serialize(message *Message) ([]byte, error) {
	builder := flatbuffers.NewBuilder(16)
	payload := message.Payload()
	lPayload := len(payload)
	FBMessage.MessageStartPayloadVector(builder, lPayload)
	for i := range payload {
		builder.PrependByte(payload[lPayload-i-1])
	}
	payloadOffset := builder.EndVector(lPayload)
	idOffset := builder.CreateString(message.Id())
	fromOffset := builder.CreateString(message.From())
	toOffset := builder.CreateString(message.To())
	uriOffset := builder.CreateString(message.Uri())
	FBMessage.MessageStart(builder)
	FBMessage.MessageAddId(builder, idOffset)
	FBMessage.MessageAddFrom(builder, fromOffset)
	FBMessage.MessageAddTo(builder, toOffset)
	FBMessage.MessageAddUri(builder, uriOffset)
	FBMessage.MessageAddMessageType(builder, (int32)(message.messageType))
	FBMessage.MessageAddPayload(builder, payloadOffset)
	offset := FBMessage.MessageEnd(builder)
	builder.Finish(offset)
	return builder.Bytes[builder.Head():], nil
}

func (p *FBMessageParser) Deserialize(buffer []byte) (*Message, error) {
	if len(buffer) < 1 {
		return nil, errors.New("invalid buffer format")
	}
	fbMessage := FBMessage.GetRootAsMessage(buffer, 0)
	id := (string)(fbMessage.Id())
	from := (string)(fbMessage.From())
	to := (string)(fbMessage.To())
	uri := (string)(fbMessage.Uri())
	msgType := fbMessage.MessageType()
	payload := fbMessage.Payload()
	return NewMessage(id, from, to, uri, (int)(msgType), payload), nil
}
