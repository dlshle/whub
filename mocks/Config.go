package mocks

var TestMode bool

func init() {
	TestMode = false
}

func StartTestEnv() {
	TestMode = true
	StartServer()
	StartClient()
}
