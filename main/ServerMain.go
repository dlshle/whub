package main

import (
	"fmt"
	"whub/hub_common/roles"
	"whub/hub_server"
)

func ServerTest() {
	role := roles.NewServer("test", "xx", "0.0.0.0", 1234)
	s := hub_server.NewServer(role, "")
	e := s.Start()
	if e != nil {
		fmt.Println(e)
	}
}
