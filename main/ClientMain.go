package main

import (
	"fmt"
	"time"
	"wsdk/common/connection"
	"wsdk/relay_client"
	"wsdk/relay_client/services"
	base_conn "wsdk/relay_common/connection"
	"wsdk/relay_common/messages"
)

func ClientTest() {
	c := relay_client.NewClient(connection.TypeWS, "192.168.0.182", 1234, base_conn.WSConnectionPath, "test1", "123456")
	err := c.Connect()
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	fmt.Println("connection done")
	resp, err := c.Request(messages.NewMessage("239845792", "aabb", "", "/service/message/broadcast", messages.MessageTypeServiceRequest, ([]byte)("asdasdasd")))
	fmt.Println("request resp, err: ", resp, err)
	resp, err = c.Request(messages.NewMessage("roleDescTest", c.Role().Id(), "", "", messages.MessageTypeClientDescriptor, ([]byte)(c.Role().Describe().String())))
	fmt.Println("request2 resp, err: ", resp, err)

	httpSvc := new(services.HTTPClientService)
	registerSvc(c, httpSvc)
	time.Sleep(70 * time.Second)
	fmt.Println("client timeout done")
}

func RunMultipleClientTest(n int) {
	for i := 0; i < n; i++ {
		go ClientTest()
	}
}

func registerSvc(c *relay_client.Client, svc relay_client.IClientService) {
	c.SetService(svc)
	err := c.RegisterService()
	if err != nil {
		fmt.Println("register service error: ", err)
	} else {
		err = c.StartService()
		if err != nil {
			fmt.Println("start service error: ", err)
		}
	}
}
