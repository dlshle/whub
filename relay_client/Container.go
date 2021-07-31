package relay_client

import "wsdk/common/ioc"

var Container *ioc.Container

func init() {
	Container = ioc.New()
}
