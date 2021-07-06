package main

import "time"

func main() {
	go ServerTest()
	time.Sleep(time.Second)
	go ClientTest()

	time.Sleep(time.Minute * 10)
}
