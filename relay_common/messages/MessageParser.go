package messages

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"wsdk/relay_common/utils"
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
		id = (string)(serialMessage[0:to+1])
	}
	stageMap[1] = func(from, to int) {
		msgFrom = (string)(serialMessage[from: to+1])
	}
	stageMap[2] = func(from, to int) {
		msgTo = (string)(serialMessage[from: to+1])
	}
	stageMap[3] = func(from, to int) {
		msgUri = (string)(serialMessage[from: to+1])
	}
	stageMap[4] = func(from, to int) {
		msgType, err = strconv.Atoi((string)(serialMessage[from: to+1]))
		if err != nil {
			hasError = true
		}
	}
	stageMap[5] = func(from, to int) {
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
	return NewMessage(id, msgFrom, msgTo, msgUri, msgType, payload), nil
}

type StreamMessageParser struct{}

const (
	StreamParserHeaderShiftID = 5
	StreamParserHeaderShiftFrom = 4
	StreamParserHeaderShiftTo = 3
	StreamParserHeaderShiftUri = 2
	StreamParserHeaderShiftType = 1
	StreamParserHeaderShiftPayload = 0
)

func assembleBitData(bit uint8, position uint8) uint8 {
	if position > 7 {
		return 0
	}
	return bit << position
}

// bit from low to high
func getBitAt(data, bit uint8) uint8 {
	if bit > 7 {
		return 0
	}
	return data & (1 << bit)
}

func assembleHeaderFrom(message *Message) uint8 {
	// header ID FROM TO URI TYPE PAYLOAD
	var header uint8 = 0
	var payloadBit uint8 = 0
	if message.Payload() != nil {
		payloadBit = 1
	}
	msgPropDataList := []uint8{
		assembleBitData(utils.GetStringBitLen(message.Id()), StreamParserHeaderShiftID),
		assembleBitData(utils.GetStringBitLen(message.From()), StreamParserHeaderShiftFrom),
		assembleBitData(utils.GetStringBitLen(message.To()), StreamParserHeaderShiftTo),
		assembleBitData(utils.GetStringBitLen(message.Uri()), StreamParserHeaderShiftUri),
		assembleBitData(1, StreamParserHeaderShiftType),
		assembleBitData(payloadBit, StreamParserHeaderShiftPayload),
	}
	for _, data := range msgPropDataList {
		header |= data
	}
	return header
}

func assembleSmallStringLengthData(header uint8, bit uint8, data string) byte {
	if getBitAt(header, bit) == 0 {
		return 0
	}
	return (byte)(len(data))
}

func assembleLongStreamLengthData(header uint8, bit uint8, data []byte) []byte {
	if getBitAt(header, bit) == 0 {
		return nil
	}
	length := uint32(len(data))
	lengthInBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(make([]byte, 4), length)
	return lengthInBytes
}

func doIfInHeader(header uint8, bit uint8, cb func()) {
	if getBitAt(header, bit) == 1 {
		cb()
	}
}

func assembleLengthDataFrom(header uint8, message *Message) []byte {
	buffer := bytes.Buffer{}
	initDataLengthList := []uint8 {
		assembleSmallStringLengthData(header, StreamParserHeaderShiftID, message.Id()),
		assembleSmallStringLengthData(header, StreamParserHeaderShiftFrom, message.From()),
		assembleSmallStringLengthData(header, StreamParserHeaderShiftTo, message.To()),
	}
	buffer.Write(initDataLengthList)
	doIfInHeader(header, StreamParserHeaderShiftUri, func() {
		buffer.Write(assembleLongStreamLengthData(header, StreamParserHeaderShiftUri, ([]byte)(message.Uri())))
	})
	doIfInHeader(header, StreamParserHeaderShiftPayload, func() {
		buffer.Write(assembleLongStreamLengthData(header, StreamParserHeaderShiftUri, message.Payload()))
	})
	return buffer.Bytes()
}

func assembleContentDataFrom(header uint8, message *Message) []byte {
	buffer := bytes.Buffer{}
	doIfInHeader(header, StreamParserHeaderShiftID, func() {
		buffer.WriteString(message.Id())
	})
	doIfInHeader(header, StreamParserHeaderShiftFrom, func() {
		buffer.WriteString(message.From())
	})
	doIfInHeader(header, StreamParserHeaderShiftTo, func() {
		buffer.WriteString(message.To())
	})
	doIfInHeader(header, StreamParserHeaderShiftUri, func() {
		buffer.WriteString(message.Uri())
	})
	// type
	buffer.WriteByte((byte)(message.MessageType()))
	doIfInHeader(header, StreamParserHeaderShiftPayload, func() {
		buffer.Write(message.Payload())
	})
	return buffer.Bytes()
}

func (p *StreamMessageParser) Serialize(message *Message) ([]byte, error) {
	if utils.GetStringBitLen(message.Id()) == 0 {
		return nil, errors.New("invalid message: message has no id")
	}
	// existence of ? fields depends on header value
	// header || len_section || data_section
	// header | (id_len | from_len? | to_len? | uri_len(4 bytes) | payload_len(4 bytes)) | (id | from? | to? | uri? | type | payload)
	buffer := bytes.Buffer{}
	header := assembleHeaderFrom(message)
	buffer.WriteByte(header)
	buffer.Write(assembleLengthDataFrom(header, message))
	buffer.Write(assembleContentDataFrom(header, message))
	return buffer.Bytes(), nil
}

func byteArrBoundCheck(arr []byte, from, ext int) error {
	if len(arr) > from + ext {
		return nil
	}
	return errors.New("invalid message stream format: insufficient stream length")
}

func computeLengthDataFrom(header uint8, stream []byte, counter *int) (data []int, err error) {
	lengthInfo := make([]int, 5)
	byteCounter := *counter
	getNextLengthDataIfExist := func(bit uint8) error {
		doIfInHeader(header, bit, func() {
			lengthInfo = append(lengthInfo, (int)(stream[byteCounter]))
			byteCounter++
			if len(stream) <= byteCounter {
				err = errors.New("invalid stream message format: insufficient stream length")
			}
		})
		return err
	}
	for _, bit := range []uint8 {
		StreamParserHeaderShiftID,
		StreamParserHeaderShiftFrom,
		StreamParserHeaderShiftTo,
	} {
		if getNextLengthDataIfExist(bit) != nil {
			return
		}
	}
	doIfInHeader(header, StreamParserHeaderShiftUri, func() {
		if len(stream) <= byteCounter + 4 {
			err = errors.New("invalid stream message format: insufficient stream length")
			return
		}
		length := binary.LittleEndian.Uint32(stream[byteCounter:byteCounter+4])
		lengthInfo = append(lengthInfo, (int)(length))
		byteCounter += 4
	})
	if err != nil {
		return
	}
	doIfInHeader(header, StreamParserHeaderShiftPayload, func() {
		if len(stream) <= byteCounter + 4 {
			err = errors.New("invalid stream message format: insufficient stream length")
			return
		}
		length := binary.LittleEndian.Uint32(stream[byteCounter:byteCounter+4])
		lengthInfo = append(lengthInfo, (int)(length))
		byteCounter += 4
	})
	if err != nil {
		return
	}
	data = lengthInfo
	*counter = byteCounter
	return
}

func computeMessageFrom(header uint8, lengthInfo []int, data []byte, counter *int) (msg *Message, err error) {
	// assemble message
	byteCounter := *counter
	lengthCounter := 0
	var id, from, to, uri string
	var msgType int
	var payload []byte
	makeWithErrIfInHeader := func(h uint8, bit uint8, cb func()) func() error {
		return func() error {
			doIfInHeader(h, bit, cb)
			return err
		}
	}
	err = utils.ProcessWithError([]func() error{
		makeWithErrIfInHeader(header, StreamParserHeaderShiftID, func() {
			length := lengthInfo[lengthCounter]
			if err = byteArrBoundCheck(data, byteCounter, length); err != nil {
				return
			}
			id = (string)(data[byteCounter: byteCounter + length])
			byteCounter += length
			lengthCounter++
		}),
		makeWithErrIfInHeader(header, StreamParserHeaderShiftFrom, func() {
			length := lengthInfo[lengthCounter]
			if err = byteArrBoundCheck(data, byteCounter, length); err != nil {
				return
			}
			from = (string)(data[byteCounter: byteCounter + length])
			byteCounter += length
			lengthCounter++
		}),
		makeWithErrIfInHeader(header, StreamParserHeaderShiftTo, func() {
			length := lengthInfo[lengthCounter]
			if err = byteArrBoundCheck(data, byteCounter, length); err != nil {
				return
			}
			to = (string)(data[byteCounter: byteCounter + length])
			byteCounter += length
			lengthCounter++
		}),
		makeWithErrIfInHeader(header, StreamParserHeaderShiftType, func() {
			if err = byteArrBoundCheck(data, byteCounter, 1); err != nil {
				return
			}
			msgType = (int)(data[byteCounter])
			byteCounter++
		}),
		makeWithErrIfInHeader(header, StreamParserHeaderShiftTo, func() {
			length := lengthInfo[lengthCounter]
			if err = byteArrBoundCheck(data, byteCounter, length); err != nil {
				return
			}
			uri = (string)(data[byteCounter: byteCounter + length])
			byteCounter += length
			lengthCounter++
		}),
		makeWithErrIfInHeader(header, StreamParserHeaderShiftPayload, func() {
			length := lengthInfo[lengthCounter]
			if err = byteArrBoundCheck(data, byteCounter, length); err != nil {
				return
			}
			payload = data[byteCounter: byteCounter + length]
			byteCounter += length
			lengthCounter++
		}),
	})
	if err != nil {
		return
	}
	msg = NewMessage(id, from, to, uri, msgType, payload)
	return
}

func (p *StreamMessageParser) Deserialize(stream []byte) (msg *Message, err error) {
	if len(stream) < 3 {
		return nil, errors.New("invalid message format(insufficient stream length)")
	}
	header := stream[0]
	byteCounter := 1
	lengthData, err := computeLengthDataFrom(header, stream[1:], &byteCounter)
	if err != nil {
		return
	}
	return computeMessageFrom(header, lengthData, stream, &byteCounter)
}