package services

// tcp := app.Command("tcp", "proxy on tcp mode")
// t := tcp.Flag("tcp-timeout", "tcp timeout milliseconds when connect to real server or parent proxy").Default("2000").Int()

const (
	TYPE_TCP     = "tcp"
	TYPE_UDP     = "udp"
	TYPE_HTTP    = "http"
	TYPE_TLS     = "tls"
	TYPE_SOCKS   = "socks"
	CONN_CONTROL = uint8(1)
	CONN_SERVER  = uint8(2)
	CONN_CLIENT  = uint8(3)
)

type Args struct {
	Local     *string
	CertBytes []byte
	KeyBytes  []byte
}

type TunnelServerArgs struct {
	Args
	IsUDP   *bool
	Key     *string
	Timeout *int
}

type TunnelClientArgs struct {
	Args
	IsUDP   *bool
	Key     *string
	Timeout *int
}

type TunnelBridgeArgs struct {
	Args
	Timeout *int
}

type TCPArgs struct {
	Args
	ParentType          *string
	IsTLS               *bool
	Timeout             *int
	PoolSize            *int
	CheckParentInterval *int
}

type HTTPArgs struct {
	Args
	Always              *bool
	HTTPTimeout         *int
	Interval            *int
	Blocked             *string
	Direct              *string
	AuthFile            *string
	Auth                *[]string
	ParentType          *string
	LocalType           *string
	Timeout             *int
	PoolSize            *int
	CheckParentInterval *int
}

type SOCKSArgs struct {
	Args
	Always              *bool
	HTTPTimeout         *int
	Interval            *int
	Blocked             *string
	Direct              *string
	AuthFile            *string
	Auth                *[]string
	ParentType          *string
	LocalType           *string
	Timeout             *int
	PoolSize            *int
	CheckParentInterval *int
}

type UDPArgs struct {
	Args
	ParentType          *string
	Timeout             *int
	PoolSize            *int
	CheckParentInterval *int
}

func (a *TCPArgs) Protocol() string {
	if *a.IsTLS {
		return "tls"
	}
	return "tcp"
}

const (
	SOCKS5_VERSION = 0x05

	// Authentication methods
	SOCKS5_AUTH_NONE      = 0x00
	SOCKS5_AUTH_PASSWORD  = 0x02
	SOCKS5_AUTH_NO_ACCEPT = 0xFF

	// Commands
	SOCKS5_CMD_CONNECT = 0x01
	SOCKS5_CMD_BIND    = 0x02
	SOCKS5_CMD_UDP     = 0x03

	// Address types
	SOCKS5_ATYP_IPV4   = 0x01
	SOCKS5_ATYP_DOMAIN = 0x03
	SOCKS5_ATYP_IPV6   = 0x04

	// Reply codes
	SOCKS5_REP_SUCCESS            = 0x00
	SOCKS5_REP_GENERAL_FAILURE    = 0x01
	SOCKS5_REP_CONN_NOT_ALLOWED   = 0x02
	SOCKS5_REP_NET_UNREACHABLE    = 0x03
	SOCKS5_REP_HOST_UNREACHABLE   = 0x04
	SOCKS5_REP_CONN_REFUSED       = 0x05
	SOCKS5_REP_TTL_EXPIRED        = 0x06
	SOCKS5_REP_CMD_NOT_SUPPORTED  = 0x07
	SOCKS5_REP_ATYP_NOT_SUPPORTED = 0x08
)
