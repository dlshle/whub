package WRCommon

type IServicePool interface {
	Get(id string) *ServiceMessage
	Add(message IServiceMessage) bool
	Remove(id string) bool
	Pull(id string) *ServiceMessage // get and remove
	KillAll()
	Cancel(id string)
	Size() int
}

// TODO implementation(need a async pool to execute the real requests)
