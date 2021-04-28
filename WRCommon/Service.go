package WRCommon

import "time"

type Service struct {
	id string
	description string
	owner *WRBaseRole
	cTime time.Time
	auth IAuth
}

type IService interface {
	Authenticate(credential ICredential)
	// TODO
}