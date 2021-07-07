package main

import (
	"fmt"
	"time"
	"wsdk/relay_client"
)

func ClientTest() {
	c := relay_client.NewClient("0.0.0.0", 1234, "aabb")
	err := c.Connect()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("connection done")
	time.Sleep(70 * time.Second)
	fmt.Println("client timeout done")
}

func RunMultipleClientTest(n int) {
	for i := 0; i < n; i++ {
		go ClientTest()
	}
}
