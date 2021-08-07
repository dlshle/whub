package connection

const (
	TypeTCP = iota
	TypeWS
	TypeUDP
	TypeRTC
	TypeHTTP
)

func IsAsyncType(connType uint8) bool {
	return connType < TypeHTTP
}
