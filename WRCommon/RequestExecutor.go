package WRCommon

type IRequestExecutor interface {
	Execute(*ServiceMessage)
}