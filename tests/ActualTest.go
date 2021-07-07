package main

import "time"

func main() {
	go ServerTest()
	time.Sleep(time.Second)
	go RunMultipleClientTest(20)

	time.Sleep(time.Minute * 10)
}
