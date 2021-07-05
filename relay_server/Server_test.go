package relay_server

import (
	"testing"
	"time"
	"wsdk/relay_common/roles"
)

func TestServer(t *testing.T) {
	role := roles.NewServer("test", "xx", "127.0.0.1", 12345)
	s := NewServer(role)
	e := s.Start()
	if e != nil {
		t.Error(e)
	}
	time.Sleep(time.Minute * 5)
}
