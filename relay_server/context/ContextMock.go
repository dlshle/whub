package context

import "wsdk/relay_common/roles"

var MockCtx *Context

func init() {
	mockCommonServer := roles.NewServer("mock", "mocked", "localhost", 1234)
	MockCtx = NewContext()
	MockCtx.Start(mockCommonServer)
}
