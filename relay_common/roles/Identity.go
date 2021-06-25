package roles

import (
	"fmt"
	"wsdk/relay_common/connection"
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

type CommonRole struct {
	id          string
	description string
	rType       int
}

type ICommonRole interface {
	Id() string
	Description() string
	Type() int
}

func NewBaseRole(id, description string, rType int) *CommonRole {
	return &CommonRole{id, description, rType}
}

func (c *CommonRole) Id() string {
	return c.id
}

func (c *CommonRole) Description() string {
	return c.description
}

func (c *CommonRole) Type() int {
	return c.rType
}

type IDescribableRole interface {
	ICommonRole
	Describe() RoleDescriptor
}

type RoleDescriptor struct {
	Id          string
	Description string
	RoleType    string
	ExtraInfo   string
	Address     string
}

func (rd RoleDescriptor) String() string {
	return fmt.Sprintf("{id:%s,description:%s,roleType:%s,extraInfo:%s}", rd.Id, rd.Description, rd.RoleType, rd.ExtraInfo)
}

type CommonClient struct {
	*connection.Connection
	*CommonRole
	pScope     int // a 16-bit
	cKey       string
	cType      int
	descriptor *RoleDescriptor
}

type ICommonClient interface {
	ICommonRole
	Scopes() []int
	HasScope(int) bool
	CKey() string
	CType() int
	Describe() RoleDescriptor
}

func (c *CommonClient) Scopes() (scopes []int) {
	scopes = make([]int, MaxPrivileges)
	for i := 0; i < MaxPrivileges; i++ {
		scopes[i] = c.pScope & i
	}
	return scopes
}

func (c *CommonClient) HasScope(scope int) bool {
	return (c.pScope & scope) != 0
}

func (c *CommonClient) CKey() string {
	return c.cKey
}

func (c *CommonClient) CType() int {
	return c.cType
}

func (c *CommonClient) Describe() RoleDescriptor {
	if c.descriptor == nil {
		c.descriptor = &RoleDescriptor{
			c.Id(),
			c.Description(),
			RoleTypeClientStr,
			fmt.Sprintf("{pScope: %d, cKey: \"%s\", cType: %d}", c.pScope, c.cKey, c.cType),
			c.Address(),
		}
	}
	return *c.descriptor
}

func NewClient(conn *connection.Connection, id string, description string, cType int, cKey string, pScope int) *CommonClient {
	return &CommonClient{Connection: conn, CommonRole: NewBaseRole(id, description, RoleTypeClient), pScope: pScope, cKey: cKey, cType: cType}
}

type CommonServer struct {
	*CommonRole
	url        string
	port       int
	descriptor *RoleDescriptor
}

type ICommonServer interface {
	IDescribableRole
	Url() string
	Port() int
}

func (s *CommonServer) Url() string {
	return s.url
}

func (s *CommonServer) Port() int {
	return s.port
}

func (s *CommonServer) Describe() RoleDescriptor {
	if s.descriptor == nil {
		s.descriptor = &RoleDescriptor{s.Id(), s.Description(), RoleTypeServerStr, s.Url(), fmt.Sprintf("%s:%s", s.Url(), s.Port())}
	}
	return *s.descriptor
}

func NewServer(id string, description string, url string, port int) *CommonServer {
	return &CommonServer{CommonRole: NewBaseRole(id, description, RoleTypeServer), url: url, port: port}
}
