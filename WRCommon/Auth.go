package WRCommon

// Maybe not so hurry?

type ICredential interface {
	Set(interface{})
	Get() string // should use jwt
	HasExpired() bool
}

type IAuth interface {
	Auth(credential ICredential) bool
}
