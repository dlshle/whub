package main

import "time"

func main() {
	go ServerTest()
	time.Sleep(time.Second)
	// go RunMultipleClientTest(20)
	go ClientTest()

	time.Sleep(time.Minute * 10)
}
