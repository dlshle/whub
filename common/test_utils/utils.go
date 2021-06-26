package test_utils

import (
	"strings"
	"sync"
	"testing"
	"time"
)

type Assertion struct {
	head         *Assertion
	id           string
	description  string
	assertion    func() bool
	shouldAssert bool
	next         *Assertion
}

type IAssertable interface {
	With(id string, description string) IAssertable
	Concurrently(id string, desc string, actions ...func()) IAssertable    // next group
	Then(id string, description string, assertion func() bool) IAssertable // next
	Cases(cases []*Assertion) IAssertable
	Do(t *testing.T)
}

func NewTestCase(id string, description string, assertion func() bool) *Assertion {
	a := &Assertion{
		id:           id,
		description:  description,
		assertion:    assertion,
		shouldAssert: true,
	}
	a.head = a
	return a
}

func NewTestGroup(id string, description string) IAssertable {
	a := &Assertion{
		id:          id,
		description: description,
	}
	a.head = a
	return a
}

func (a *Assertion) With(id string, description string) IAssertable {
	a.next = &Assertion{
		head:         a.head,
		id:           id,
		description:  description,
		shouldAssert: false,
	}
	return a.next
}

func (a *Assertion) Concurrently(id string, desc string, actions ...func()) IAssertable {
	actionFunc := func() bool {
		var wg sync.WaitGroup
		for _, act := range actions {
			wg.Add(1)
			go (func(action func()) {
				action()
				wg.Done()
			})(act)
		}
		wg.Wait()
		return true
	}
	a.next = &Assertion{
		head:         a.head,
		id:           id,
		description:  desc,
		assertion:    actionFunc,
		shouldAssert: false,
	}
	return a.next
}

func (a *Assertion) Then(id string, description string, assertion func() bool) IAssertable {
	a.next = &Assertion{
		head:         a.head,
		id:           id,
		description:  description,
		assertion:    assertion,
		shouldAssert: true,
	}
	return a.next
}

func (a *Assertion) Cases(cases []*Assertion) IAssertable {
	curr := a
	for _, c := range cases {
		if c != nil {
			curr.next = c
			c.head = curr.head
			curr = c
		}
	}
	return curr
}

func indents(level int) string {
	if level == 0 {
		return ""
	}
	builder := strings.Builder{}
	for level > 0 {
		builder.WriteByte(' ')
		level--
	}
	return builder.String()
}

func (a *Assertion) Do(t *testing.T) {
	startTiem := time.Now()
	curr := a.head
	indent := 0
	for curr != nil {
		if curr.shouldAssert {
			t.Logf("%sRunning case %s[%s]\n", indents(indent), curr.id, curr.description)
		} else {
			t.Logf("%sRunning operation %s[%s]\n", indents(indent), curr.id, curr.description)
		}
		if curr.assertion != nil {
			if curr.shouldAssert {
				assertCase(t, indent, curr.id, curr.description, curr.assertion)
			} else {
				curr.assertion()
			}
		} else {
			indent += 2
		}
		curr = curr.next
	}
	t.Log("All test finished, overall runtime: ", time.Now().Sub(startTiem))
}

func assertCase(t *testing.T, indent int, id string, desc string, assertion func() bool) {
	if assertion() {
		t.Logf("%s✅ %s passed\n", indents(indent), id)
	} else {
		t.Errorf("%s❌ %s(%s) failed\n", indents(indent), id, desc)
	}
}
