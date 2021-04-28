package WRCommon

type ICredential interface {
	Set(interface{})
	Get() string
	HasExpired() bool
}

type IAuth interface {
	Auth(credential ICredential) bool
}
