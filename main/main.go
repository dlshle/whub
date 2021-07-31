package main

import "time"

func main() {
	role := "server"
	if role == "server" {
		ServerTest()
	} else {
		go RunMultipleClientTest(1)
	}
	time.Sleep(time.Minute * 10)
}
