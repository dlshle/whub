package main

import (
	"fmt"
	"wsdk/relay_common/roles"
	"wsdk/relay_server"
)

func ServerTest() {
	role := roles.NewServer("test", "xx", "0.0.0.0", 1234)
	s := relay_server.NewServer(role, "")
	e := s.Start()
	if e != nil {
		fmt.Println(e)
	}
}
