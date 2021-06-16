package messages

import (
	"testing"
	"wsdk/gommon/test_utils"
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
	}).Do(t)
}
