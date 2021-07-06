package main

import (
	"fmt"
	"wsdk/relay_client"
)

func ClientTest() {
	c := relay_client.NewClient("0.0.0.0", 1234, "aabb")
	err := c.Connect()
	if err != nil {
		fmt.Println(err)
	}
}
