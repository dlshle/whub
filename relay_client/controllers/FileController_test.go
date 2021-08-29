package controllers

import "testing"

func TestFileController(t *testing.T) {
	c, e := NewFileController("./", 512)
	if e != nil {
		t.Fatal(e.Error())
		return
	}
	info, err := c.List(".")
	if err != nil {
		t.Fatal(e.Error())
		return
	}
	t.Log(info)

	data, err := c.Read("FileController.go", 0)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log((string)(data))
}
