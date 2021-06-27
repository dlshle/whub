package ioc

import (
	"testing"
	"wsdk/common/test_utils"
)

type Box struct {
	name string
}

type Jar struct {
	liter int
}

type Sealed struct {
	box Box
	jar Jar
}

type TSealed struct {
	b *Box `$autowire`
	j *Jar `$autowire`
}

func (t *TSealed) SetB(b *Box) {
	t.b = b
}

func (t *TSealed) SetJ(j *Jar) {
	t.j = j
}

type Assembled struct {
	headBox Box
	sealed  Sealed
}

func TestContainer(t *testing.T) {
	c := NewContainer()

	test_utils.NewTestGroup("Container", "container tests").Cases([]*test_utils.Assertion{
		test_utils.NewTestCase("basic injection", "", func() bool {
			c.RegisterComponent("box-1", &Box{name: "box-1"})
			c.RegisterComponent("jar-1", &Jar{liter: 1})
			type TSealed struct {
				*Box `$inject:box-1`
				*Jar `$inject:jar-1`
			}
			tSealed := &TSealed{}
			err := c.InjectFields(tSealed)
			if err != nil {
				t.Log(err)
				return false
			}
			t.Log(tSealed.Box, tSealed.Jar)
			return tSealed.Box.name == "box-1" && tSealed.Jar.liter == 1
		}),
		test_utils.NewTestCase("basic autowiring", "", func() bool {
			c.Clear()
			c.AutoRegister(&Box{name: "box"})
			c.AutoRegister(&Jar{liter: 1})
			type TSealed struct {
				*Box `$autowire`
				*Jar `$autowire`
			}
			tSealed := &TSealed{}
			// t.Log(reflect.TypeOf(tSealed).Elem())
			err := c.AutoWireFieldsByType(tSealed)
			if err != nil {
				t.Log(err)
				return false
			}
			t.Log(tSealed.Box, tSealed.Jar)
			return tSealed.Box.name == "box" && tSealed.Jar.liter == 1
		}),
		test_utils.NewTestCase("public setter test", "", func() bool {
			tSealed := &TSealed{}
			// t.Log(reflect.TypeOf(tSealed).Elem())
			err := c.AutoWireFieldsByType(tSealed)
			if err != nil {
				t.Log(err)
				return false
			}
			t.Log(tSealed.b, tSealed.j)
			return tSealed.b.name == "box" && tSealed.j.liter == 1
		}),
	}).Do(t)
}
