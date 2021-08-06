package container

import "wsdk/common/ioc"

var Container *ioc.Container

func init() {
	Container = ioc.New()
}
