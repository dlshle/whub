package uri

import (
	"fmt"
	"testing"
)

func TestNewUriTree(test *testing.T) {
	t := NewUriTree()
	p0 := "a/b/:p00/c/:p01"
	p1 := "a/:p1/b/*any"
	pc0 := "x/y/z"
	pReplicate := "a/:p1"
	handler := func(params map[string]string, qp map[string]string) error {
		fmt.Println("handler exec with ", params, qp)
		return nil
	}
	e := t.Add(p0, handler, false)
	if e != nil {
		test.Error(p0, e)
		return
	}
	e = t.Add(p1, handler, false)
	if e != nil {
		test.Error(p1, e)
		return
	}

	e = t.Add(pc0, handler, false)
	if e != nil {
		test.Error(p1, e)
		return
	}

	// test with replicate params
	e = t.Add(pReplicate, handler, false)
	if e == nil {
		test.Error("replicated params should report error!")
		return
	} else {
		fmt.Println(e.Error(), " PASS")
	}

	e = t.FindAndHandle("a/b/~/c/~")
	e = t.FindAndHandle("a/b/~/c/~?q0=0&q1=1")
	e = t.FindAndHandle(pc0 + "?q=1,2,3&p=4,5,6")

	// incorrect query param formats
	e = t.FindAndHandle(pc0 + "?t&&")
	if e == nil {
		test.Error("should report error on incorrect query param format ?t&&")
	}

	e = t.FindAndHandle(pc0 + "?t&??&")
	if e == nil {
		test.Error("should report error on incorrect query param format ?t&&")
	}

	e = t.FindAndHandle(pc0 + "?t==2")
	if e == nil {
		test.Error("should report error on incorrect query param format ?t&&")
	}

	e = t.FindAndHandle(pc0 + "?t=2=3")
	if e == nil {
		test.Error("should report error on incorrect query param format ?t&&")
	}

	e = t.FindAndHandle(pc0 + "?t=2?")
	if e == nil {
		test.Error("should report error on incorrect query param format ?t&&")
	}
}
