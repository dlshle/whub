package main

import "time"

func main() {
	go ServerTest()
	// go RunMultipleClientTest(20)
	go RunMultipleClientTest(200)

	time.Sleep(time.Minute * 10)
}
