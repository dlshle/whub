package WRCommon

import (
	"wsdk/WRCommon/Connection"
	"wsdk/WRCommon/Message"
	"wsdk/WRCommon/Utils"
)

// Role types
const (
	RoleTypeClient = 0
	RoleTypeServer = 1
)

const (
	RoleTypeClientStr = "Client"
	RoleTypeServerStr = "Server"
)

// Client types
const (
	ClientTypeAnonymous     = 0
	ClientTypeAuthenticated = 1
	ClientTypeRoot          = 2
)

// Authenticated privileges
// 16-bit binary
const MaxPrivileges = 5
const (
	PRMessage = 0b00001
	PWMessage = 0b00010
	// below privileges depends on 0x1 and 0x2
	PDiscoverClients   = 0b00100
	PReadClientDetail  = 0b01000
	PRegisterCallbacks = 0b10000
)

type WRBaseRole struct {
	id          string
	description string
	rType		int
}

type IWRBaseRole interface {
	Id() string
	Description() string
	Type() int
	NewMessage(from string, msgType int, payload []byte) *Message.Message
}

func NewBaseRole(id, description string, rType int) *WRBaseRole {
	return &WRBaseRole{id, description, rType}
}

func (c *WRBaseRole) Id() string {
	return c.id
}

func (c *WRBaseRole) Description() string {
	return c.description
}

func (c *WRBaseRole) Type() int {
	return c.rType
}

func (c *WRBaseRole) NewMessage(to string, msgType int, payload []byte) *Message.Message {
	return Message.NewMessage(Utils.GenStringId(), c.Id(), to, msgType, payload)
}

type IDescribableRole interface {
	IWRBaseRole
	Describe() RoleDescriptor
}

type RoleDescriptor struct {
	Id string
	Description string
	RoleType string
	ExtraInfo string
}

type WRClient struct {
	conn *Connection.WRConnection
	*WRBaseRole
	pScope int // a 16-bit
	cKey   string
	cType int
	descriptor *RoleDescriptor
}

type IWRClient interface {
	Scopes() []int
	HasScope(int) bool
	CKey() string
	CType() int
	Describe() RoleDescriptor
}

func (c *WRClient) Scopes() (scopes []int) {
	scopes = make([]int, MaxPrivileges)
	for i := 0; i < MaxPrivileges; i++ {
		scopes[i] = c.pScope & i
	}
	return scopes
}

func (c *WRClient) HasScope(scope int) bool {
	return (c.pScope & scope) != 0
}

func (c *WRClient) CKey() string {
	return c.cKey
}

func (c *WRClient) CType() int {
	return c.cType
}

func (c *WRClient) Describe() RoleDescriptor {
	if c.descriptor == nil {
		c.descriptor = &RoleDescriptor{c.Id(), c.Description(), RoleTypeClientStr, ""}
	}
	return *c.descriptor
}

func NewClient(conn *Connection.WRConnection, id string, description string, cType int, cKey string, pScope int) *WRClient {
	return &WRClient{conn: conn, WRBaseRole: NewBaseRole(id, description, RoleTypeClient), pScope: pScope, cKey: cKey, cType: cType}
}

type WRServer struct {
	*Connection.WRConnection
	*WRBaseRole
	url string
	descriptor *RoleDescriptor
}

type IWRServer interface {
	Url() string
	Describe() RoleDescriptor
}

func (s *WRServer) Url() string {
	return s.url
}

func (s *WRServer) Describe() RoleDescriptor {
	if s.descriptor == nil {
		s.descriptor = &RoleDescriptor{s.Id(), s.Description(), RoleTypeServerStr, s.Url()}
	}
	return *s.descriptor
}

func NewServer(conn *Connection.WRConnection, id string, description string, url string) *WRServer {
	return &WRServer{WRConnection: conn, WRBaseRole: NewBaseRole(id, description, RoleTypeServer), url: url}
}
