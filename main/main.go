package main

import "time"

func main() {
	role := "x"
	if role == "Server" {
		ServerTest()
	} else {
		go RunMultipleClientTest(10)
	}
	time.Sleep(time.Minute * 10)
}
