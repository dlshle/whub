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

	size := t.Size()

	e = t.Remove(p0)
	if e != nil {
		test.Errorf("should successfully remove the path w/o any err. err: %s", e.Error())
	}
	if t.Size() != size-1 {
		test.Error("size did not decrement after removing a path")
	}

	t = NewUriTree()
	t.Add(p0, handler, false)
	t.Add(p1, handler, false)
	// size == 2
	for i := 0; i < 14; i++ {
		t.Add((string)('a'+i), handler, false)
	}
	for k, _ := range t.constPathMap {
		test.Logf("constKey: %s\n", k)
	}
	if t.Size() != 16 {
		test.Error("Size not match")
	}
	if t.unCompactedLeaves.Size() != 0 {
		test.Error("compact was not conducted after adding 16 paths")
	}
	if len(t.constPathMap) != 14 {
		test.Error("const paths were not compacted")
	}
	e = t.FindAndHandle("a?fuck=true")
	if e != nil {
		test.Error("could not conduct query with a?fuck=true")
	}
}
