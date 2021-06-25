package messages

import (
	"testing"
	"wsdk/common/test_utils"
)

func TestFBMessageParser(t *testing.T) {
	p := NewFBMessageParser()
	tg := test_utils.NewTestGroup("StreamMessageParser", "")
	tg.Cases([]*test_utils.Assertion{
		test_utils.NewTestCase("Serialization/Deserialization", "", func() bool {
			m0 := NewMessage("1", "a", "t", "x/y/z", MessageTypeACK, ([]byte)("hello"))
			serialized, err := p.Serialize(m0)
			t.Log(serialized)
			if err != nil {
				t.Log("Serialization failed due to ", err)
				return false
			}
			m, err := p.Deserialize(serialized)
			if err != nil {
				t.Log("Deserialization failed due to ", err)
				return false
			}
			return m.Equals(m0)
		}),
		test_utils.NewTestCase("Partial", "", func() bool {
			m0 := &Message{id: "ppp", uri: "ok"}
			s, e := p.Serialize(m0)
			if e != nil {
				t.Log("Serialization failed due to ", e)
				return false
			}
			t.Log(s)
			m, e := p.Deserialize(s)
			if e != nil {
				t.Log("Deserialization failed due to ", e)
				return false
			}
			t.Log("Deserialized ", m.String())
			return m.Equals(m0)
		}),
	}).Do(t)
}
