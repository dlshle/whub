package roles

import (
	"fmt"
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
	ClientTypeManager       = 2
	ClientTypeRoot          = 3
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
	SetDescription(description string)
	Type() int
	SetType(rtype int)
}

func NewCommonRole(id, description string, rType int) *CommonRole {
	return &CommonRole{id, description, rType}
}

func (c *CommonRole) Id() string {
	return c.id
}

func (c *CommonRole) Description() string {
	return c.description
}

func (c *CommonRole) SetDescription(description string) {
	c.description = description
}

func (c *CommonRole) Type() int {
	return c.rType
}

func (c *CommonRole) SetType(rtype int) {
	c.rType = rtype
}

type IDescribableRole interface {
	ICommonRole
	Describe() RoleDescriptor
}

type RoleDescriptor struct {
	Id          string `json:"id"`
	Description string `json:"description"`
	RoleType    string `json:"roleType"`
	ExtraInfo   string `json:"extraInfo"`
	Address     string `json:"address"`
}

func (rd RoleDescriptor) String() string {
	return fmt.Sprintf("{\"id\":\"%s\",\"description\":\"%s\",\"roleType\":\"%s\",\"address\":\"%s\",\"extraInfo\":\"%s\"}", rd.Id, rd.Description, rd.RoleType, rd.Address, rd.ExtraInfo)
}

type CommonClient struct {
	*CommonRole
	pScope     int // a 16-bit
	cKey       string
	cType      int
	descriptor *RoleDescriptor
}

type ClientExtraInfoDescriptor struct {
	PScope int    `json:"pScope"`
	CKey   string `json:"cKey"`
	CType  int    `json:"cType"`
}

type ICommonClient interface {
	ICommonRole
	SetPScope(pscope int)
	Scopes() []int
	HasScope(int) bool
	CKey() string
	SetCKey(ckey string)
	CType() int
	SetCType(ctype int)
	Describe() RoleDescriptor
}

func (c *CommonClient) Scopes() (scopes []int) {
	scopes = make([]int, MaxPrivileges)
	for i := 0; i < MaxPrivileges; i++ {
		scopes[i] = c.pScope & i
	}
	return scopes
}

func (c *CommonClient) PScope() int {
	return c.pScope
}

func (c *CommonClient) HasScope(scope int) bool {
	return (c.pScope & scope) != 0
}

func (c *CommonClient) SetPScope(pscope int) {
	c.pScope = pscope
}

func (c *CommonClient) CKey() string {
	return c.cKey
}

func (c *CommonClient) SetCKey(ckey string) {
	c.cKey = ckey
}

func (c *CommonClient) CType() int {
	return c.cType
}

func (c *CommonClient) SetCType(ctype int) {
	c.cType = ctype
}

func (c *CommonClient) Describe() RoleDescriptor {
	if c.descriptor == nil {
		c.descriptor = &RoleDescriptor{
			c.Id(),
			c.Description(),
			RoleTypeClientStr,
			fmt.Sprintf("{\\\"pScope\\\": %d, \\\"cKey\\\": \\\"%s\\\", \\\"cType\\\": %d}", c.pScope, c.cKey, c.cType),
			"",
		}
	}
	return *c.descriptor
}

func NewClient(id string, description string, cType int, cKey string, pScope int) *CommonClient {
	return &CommonClient{CommonRole: NewCommonRole(id, description, RoleTypeClient), pScope: pScope, cKey: cKey, cType: cType}
}

func NewClientByDescriptor(descriptor RoleDescriptor, infoDescriptor ClientExtraInfoDescriptor) *CommonClient {
	return NewClient(descriptor.Id, descriptor.Description, infoDescriptor.CType, infoDescriptor.CKey, infoDescriptor.PScope)
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
	Address() string
}

func (s *CommonServer) Url() string {
	return s.url
}

func (s *CommonServer) Port() int {
	return s.port
}

func (s *CommonServer) Address() string {
	return fmt.Sprintf("%s:%d", s.Url(), s.Port())
}

func (s *CommonServer) Describe() RoleDescriptor {
	if s.descriptor == nil {
		s.descriptor = &RoleDescriptor{s.Id(), s.Description(), RoleTypeServerStr, s.Url(), fmt.Sprintf("%s:%d", s.Url(), s.Port())}
	}
	return *s.descriptor
}

func NewServer(id string, description string, url string, port int) *CommonServer {
	return &CommonServer{CommonRole: NewCommonRole(id, description, RoleTypeServer), url: url, port: port}
}
