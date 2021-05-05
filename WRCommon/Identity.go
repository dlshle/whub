package WRCommon

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
	cType       int
}

type WRClient struct {
	*WRConnection
	*WRBaseRole
	pScope int // a 16-bit
	cKey   string
}

func (c *WRClient) Id() string {
	return c.id
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

func (c *WRClient) Description() string {
	return c.description
}

func (c *WRClient) CKey() string {
	return c.cKey
}

func (c *WRClient) Type() int {
	return c.cType
}

func NewAnonymousClient(conn *WRConnection) *WRClient {
	return NewClient(conn, "", "", ClientTypeAnonymous, "", PRMessage)
}

func NewClient(conn *WRConnection, id string, description string, cType int, cKey string, pScope int) *WRClient {
	return &WRClient{conn, &WRBaseRole{id, description, cType}, pScope, cKey}
}
