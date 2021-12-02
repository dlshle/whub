package container

import "whub/common/ioc"

var Container *ioc.Container

func init() {
	Container = ioc.New()
}
