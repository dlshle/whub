package main

import (
	"fmt"
	"time"
	"wsdk/relay_client"
	"wsdk/relay_common/messages"
)

func ClientTest() {
	c := relay_client.NewClient("42.192.164.79", 1234, "aabb")
	err := c.Connect()
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	fmt.Println("connection done")
	resp, err := c.Request(messages.NewMessage("239845792", "aabb", "", "/service_manager/message/broadcast", messages.MessageTypeServiceRequest, ([]byte)("asdasdasd")))
	fmt.Println("request resp, err: ", resp, err)
	resp, err = c.Request(messages.NewMessage("roleDescTest", c.Role().Id(), "", "", messages.MessageTypeClientDescriptor, ([]byte)(c.Role().Describe().String())))
	fmt.Println("request2 resp, err: ", resp, err)
	time.Sleep(70 * time.Second)
	fmt.Println("client timeout done")
}

func RunMultipleClientTest(n int) {
	for i := 0; i < n; i++ {
		go ClientTest()
	}
}
