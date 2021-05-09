package messages

import (
	"errors"
	"fmt"
	"strconv"
)

type IMessageHandler interface {
	Serialize(message *Message) ([]byte, error)
	Deserialize([]byte) (*Message, error)
}

type SimpleMessageHandler struct{}

func NewSimpleMessageHandler() *SimpleMessageHandler {
	return &SimpleMessageHandler{}
}

func (h *SimpleMessageHandler) Serialize(message *Message) ([]byte, error) {
	return ([]byte)(fmt.Sprintf("%s*%s*%s*%d*%s", message.Id(), message.From(), message.To(), message.MessageType(), message.Payload())), nil
}

func (h *SimpleMessageHandler) Deserialize(serialMessage []byte) (msg *Message, err error) {
	last := 0
	stage := 0
	size := len(serialMessage)
	lastIndex := 4
	var id, msgFrom, msgTo string
	var msgType int
	var payload []byte
	hasError := false
	stageMap := make(map[int]func(int, int))
	stageMap[0] = func(from, to int) {
		id = (string)(serialMessage[0:to+1])
	}
	stageMap[1] = func(from, to int) {
		msgFrom = (string)(serialMessage[from: to+1])
	}
	stageMap[2] = func(from, to int) {
		msgTo = (string)(serialMessage[from: to+1])
	}
	stageMap[3] = func(from, to int) {
		msgType, err = strconv.Atoi((string)(serialMessage[from: to+1]))
		if err != nil {
			hasError = true
		}
	}
	stageMap[4] = func(from, to int) {
		payload = serialMessage[from: size]
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
	return NewMessage(id, msgFrom, msgTo, msgType, payload), nil
}
