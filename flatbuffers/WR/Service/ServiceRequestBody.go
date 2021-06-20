// Code generated by the FlatBuffers compiler. DO NOT EDIT.

package Service

import (
	flatbuffers "github.com/google/flatbuffers/go"
)

type ServiceRequestBody struct {
	_tab flatbuffers.Table
}

func GetRootAsServiceRequestBody(buf []byte, offset flatbuffers.UOffsetT) *ServiceRequestBody {
	n := flatbuffers.GetUOffsetT(buf[offset:])
	x := &ServiceRequestBody{}
	x.Init(buf, n+offset)
	return x
}

func (rcv *ServiceRequestBody) Init(buf []byte, i flatbuffers.UOffsetT) {
	rcv._tab.Bytes = buf
	rcv._tab.Pos = i
}

func (rcv *ServiceRequestBody) Table() flatbuffers.Table {
	return rcv._tab
}

func (rcv *ServiceRequestBody) Uri() []byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(4))
	if o != 0 {
		return rcv._tab.ByteVector(o + rcv._tab.Pos)
	}
	return nil
}

func (rcv *ServiceRequestBody) Method() int8 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(6))
	if o != 0 {
		return rcv._tab.GetInt8(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *ServiceRequestBody) MutateMethod(n int8) bool {
	return rcv._tab.MutateInt8Slot(6, n)
}

func (rcv *ServiceRequestBody) Data(j int) int8 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(8))
	if o != 0 {
		a := rcv._tab.Vector(o)
		return rcv._tab.GetInt8(a + flatbuffers.UOffsetT(j*1))
	}
	return 0
}

func (rcv *ServiceRequestBody) DataLength() int {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(8))
	if o != 0 {
		return rcv._tab.VectorLen(o)
	}
	return 0
}

func (rcv *ServiceRequestBody) MutateData(j int, n int8) bool {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(8))
	if o != 0 {
		a := rcv._tab.Vector(o)
		return rcv._tab.MutateInt8(a+flatbuffers.UOffsetT(j*1), n)
	}
	return false
}

func ServiceRequestBodyStart(builder *flatbuffers.Builder) {
	builder.StartObject(3)
}
func ServiceRequestBodyAddUri(builder *flatbuffers.Builder, uri flatbuffers.UOffsetT) {
	builder.PrependUOffsetTSlot(0, flatbuffers.UOffsetT(uri), 0)
}
func ServiceRequestBodyAddMethod(builder *flatbuffers.Builder, Method int8) {
	builder.PrependInt8Slot(1, Method, 0)
}
func ServiceRequestBodyAddData(builder *flatbuffers.Builder, data flatbuffers.UOffsetT) {
	builder.PrependUOffsetTSlot(2, flatbuffers.UOffsetT(data), 0)
}
func ServiceRequestBodyStartDataVector(builder *flatbuffers.Builder, numElems int) flatbuffers.UOffsetT {
	return builder.StartVector(1, numElems, 1)
}
func ServiceRequestBodyEnd(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	return builder.EndObject()
}