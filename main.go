package WSDK

import "time"

func main() {
	role := "Server"
	if role == "Server" {
		ServerTest()
	} else {
		go RunMultipleClientTest(1)
	}
	time.Sleep(time.Minute * 10)
}
