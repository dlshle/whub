package WRCommon

type IHealthCheckExecutor interface {
	DoHealthCheck() error
}
