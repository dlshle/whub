// TODO is this really necessary???
// The idea is to put this inside the message. Message should not contain URI, instead, different types of message should be identified by the messageType.
package service

// ServiceRequestMethods
const (
	GET    = 0
	POST   = 1
	PUT    = 2
	PATCH  = 3
	DELETE = 4
	HEAD   = 5
	OPTION = 6
)

type ServiceRequestBody struct {
	Uri    string
	Method byte
	Data   []byte
}
