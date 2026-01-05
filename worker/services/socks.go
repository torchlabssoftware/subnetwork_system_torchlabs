package services

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"runtime/debug"
	"strconv"

	"github.com/snail007/goproxy/manager"
	"github.com/snail007/goproxy/utils"
)

// SOCKS5 protocol constants
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

type SOCKS struct {
	outPool   utils.OutPool
	cfg       SOCKSArgs
	checker   utils.Checker
	basicAuth utils.BasicAuth
}

func (s *SOCKS) SetValidator(validator func(string, string) bool) {
	s.basicAuth.Validator = validator
}

func NewSOCKS() Service {
	return &SOCKS{
		outPool:   utils.OutPool{},
		cfg:       SOCKSArgs{},
		checker:   utils.Checker{},
		basicAuth: utils.BasicAuth{},
	}
}

func (s *SOCKS) InitService() {
	s.InitBasicAuth()
	if *s.cfg.Parent != "" {
		s.checker = utils.NewChecker(*s.cfg.Timeout, int64(*s.cfg.Interval), *s.cfg.Blocked, *s.cfg.Direct)
	}
}

func (s *SOCKS) StopService() {
	if s.outPool.Pool != nil {
		s.outPool.Pool.ReleaseAll()
	}
}

func (s *SOCKS) Start(args interface{}, validator func(string, string) bool, upstreamMgr *manager.UpstreamManager, worker *manager.Worker) (err error) {
	s.cfg = args.(SOCKSArgs)
	if *s.cfg.Parent != "" {
		log.Printf("use %s parent %s", *s.cfg.ParentType, *s.cfg.Parent)
		s.InitOutConnPool()
	}

	s.InitService()
	s.SetValidator(validator)

	host, port, _ := net.SplitHostPort(*s.cfg.Local)
	p, _ := strconv.Atoi(port)
	sc := utils.NewServerChannel(host, p)
	if *s.cfg.LocalType == TYPE_TCP {
		err = sc.ListenTCP(s.callback)
	} else {
		err = sc.ListenTls(s.cfg.CertBytes, s.cfg.KeyBytes, s.callback)
	}
	if err != nil {
		return
	}
	log.Printf("%s socks5 proxy on %s", *s.cfg.LocalType, (*sc.Listener).Addr())
	return
}

func (s *SOCKS) Clean() {
	s.StopService()
}

func (s *SOCKS) callback(inConn net.Conn) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("socks5 conn handler crashed with err : %s \nstack: %s", err, string(debug.Stack()))
		}
	}()

	// Handle SOCKS5 handshake
	err := s.handleHandshake(&inConn)
	if err != nil {
		log.Printf("socks5 handshake error from %s: %s", inConn.RemoteAddr(), err)
		utils.CloseConn(&inConn)
		return
	}

	// Handle SOCKS5 request
	address, err := s.handleRequest(&inConn)
	if err != nil {
		log.Printf("socks5 request error from %s: %s", inConn.RemoteAddr(), err)
		utils.CloseConn(&inConn)
		return
	}

	useProxy := true
	if *s.cfg.Parent == "" {
		useProxy = false
	} else if *s.cfg.Always {
		useProxy = true
	} else {
		s.checker.Add(address, true, "CONNECT", "", nil)
		useProxy, _, _ = s.checker.IsBlocked(address)
	}
	log.Printf("use proxy : %v, %s", useProxy, address)

	err = s.OutToTCP(useProxy, address, &inConn)
	if err != nil {
		if *s.cfg.Parent == "" {
			log.Printf("connect to %s fail, ERR:%s", address, err)
		} else {
			log.Printf("connect to %s parent %s fail", *s.cfg.ParentType, *s.cfg.Parent)
		}
		utils.CloseConn(&inConn)
	}
}

func (s *SOCKS) handleHandshake(inConn *net.Conn) error {
	// Read version and number of auth methods
	header := make([]byte, 2)
	if _, err := io.ReadFull(*inConn, header); err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}

	if header[0] != SOCKS5_VERSION {
		return fmt.Errorf("unsupported SOCKS version: %d", header[0])
	}

	// Read auth methods
	numMethods := int(header[1])
	methods := make([]byte, numMethods)
	if _, err := io.ReadFull(*inConn, methods); err != nil {
		return fmt.Errorf("failed to read auth methods: %w", err)
	}

	// Require username/password auth
	hasPasswordAuth := false
	for _, m := range methods {
		if m == SOCKS5_AUTH_PASSWORD {
			hasPasswordAuth = true
			break
		}
	}
	if !hasPasswordAuth {
		(*inConn).Write([]byte{SOCKS5_VERSION, SOCKS5_AUTH_NO_ACCEPT})
		return fmt.Errorf("client doesn't support password auth")
	}

	// Request password auth
	(*inConn).Write([]byte{SOCKS5_VERSION, SOCKS5_AUTH_PASSWORD})

	// Handle password auth
	if err := s.handlePasswordAuth(inConn); err != nil {
		return err
	}

	// No auth required
	//(*inConn).Write([]byte{SOCKS5_VERSION, SOCKS5_AUTH_NONE})

	return nil
}

func (s *SOCKS) handlePasswordAuth(inConn *net.Conn) error {
	// Read auth version
	header := make([]byte, 2)
	if _, err := io.ReadFull(*inConn, header); err != nil {
		return fmt.Errorf("failed to read auth header: %w", err)
	}

	// auth version should be 0x01
	if header[0] != 0x01 {
		return fmt.Errorf("unsupported auth version: %d", header[0])
	}

	// Read username
	usernameLen := int(header[1])
	username := make([]byte, usernameLen)
	if _, err := io.ReadFull(*inConn, username); err != nil {
		return fmt.Errorf("failed to read username: %w", err)
	}

	// Read password length
	passLenByte := make([]byte, 1)
	if _, err := io.ReadFull(*inConn, passLenByte); err != nil {
		return fmt.Errorf("failed to read password length: %w", err)
	}

	// Read password
	passwordLen := int(passLenByte[0])
	password := make([]byte, passwordLen)
	if _, err := io.ReadFull(*inConn, password); err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}

	// Validate credentials
	userpass := fmt.Sprintf("%s:%s", string(username), string(password))
	if !s.basicAuth.Check(userpass) {
		(*inConn).Write([]byte{0x01, 0x01}) // Auth failed
		return fmt.Errorf("authentication failed for user: %s", string(username))
	}

	log.Printf("socks5 auth success for user: %s", string(username))
	(*inConn).Write([]byte{0x01, 0x00}) // Auth success
	return nil
}

func (s *SOCKS) handleRequest(inConn *net.Conn) (string, error) {
	// Read request header: VER, CMD, RSV, ATYP
	header := make([]byte, 4)
	if _, err := io.ReadFull(*inConn, header); err != nil {
		return "", fmt.Errorf("failed to read request header: %w", err)
	}

	if header[0] != SOCKS5_VERSION {
		return "", fmt.Errorf("unsupported SOCKS version: %d", header[0])
	}

	cmd := header[1]
	atyp := header[3]

	// Only support CONNECT command
	if cmd != SOCKS5_CMD_CONNECT {
		s.sendReply(inConn, SOCKS5_REP_CMD_NOT_SUPPORTED)
		return "", fmt.Errorf("unsupported command: %d", cmd)
	}

	// Parse destination address
	var host string
	switch atyp {
	case SOCKS5_ATYP_IPV4:
		addr := make([]byte, 4)
		if _, err := io.ReadFull(*inConn, addr); err != nil {
			return "", fmt.Errorf("failed to read IPv4 address: %w", err)
		}
		host = net.IP(addr).String()

	case SOCKS5_ATYP_DOMAIN:
		// Read domain length
		lenByte := make([]byte, 1)
		if _, err := io.ReadFull(*inConn, lenByte); err != nil {
			return "", fmt.Errorf("failed to read domain length: %w", err)
		}
		domainLen := int(lenByte[0])
		domain := make([]byte, domainLen)
		if _, err := io.ReadFull(*inConn, domain); err != nil {
			return "", fmt.Errorf("failed to read domain: %w", err)
		}
		host = string(domain)

	case SOCKS5_ATYP_IPV6:
		addr := make([]byte, 16)
		if _, err := io.ReadFull(*inConn, addr); err != nil {
			return "", fmt.Errorf("failed to read IPv6 address: %w", err)
		}
		host = net.IP(addr).String()

	default:
		s.sendReply(inConn, SOCKS5_REP_ATYP_NOT_SUPPORTED)
		return "", fmt.Errorf("unsupported address type: %d", atyp)
	}

	// Read port
	portBytes := make([]byte, 2)
	if _, err := io.ReadFull(*inConn, portBytes); err != nil {
		return "", fmt.Errorf("failed to read port: %w", err)
	}
	port := binary.BigEndian.Uint16(portBytes)

	address := fmt.Sprintf("%s:%d", host, port)
	log.Printf("SOCKS5 CONNECT: %s", address)

	return address, nil
}

func (s *SOCKS) sendReply(inConn *net.Conn, rep byte) {
	// Send reply: VER, REP, RSV, ATYP, BND.ADDR, BND.PORT
	reply := []byte{SOCKS5_VERSION, rep, 0x00, SOCKS5_ATYP_IPV4, 0, 0, 0, 0, 0, 0}
	(*inConn).Write(reply)
}

func (s *SOCKS) OutToTCP(useProxy bool, address string, inConn *net.Conn) (err error) {
	inAddr := (*inConn).RemoteAddr().String()
	inLocalAddr := (*inConn).LocalAddr().String()

	var outConn net.Conn
	var _outConn interface{}
	if useProxy {
		_outConn, err = s.outPool.Pool.Get()
		if err == nil {
			outConn = _outConn.(net.Conn)
		}
	} else {
		outConn, err = utils.ConnectHost(address, *s.cfg.Timeout)
	}
	if err != nil {
		s.sendReply(inConn, SOCKS5_REP_HOST_UNREACHABLE)
		log.Printf("connect to %s , err:%s", address, err)
		utils.CloseConn(inConn)
		return
	}

	// Send success reply
	s.sendReply(inConn, SOCKS5_REP_SUCCESS)

	outAddr := outConn.RemoteAddr().String()
	outLocalAddr := outConn.LocalAddr().String()

	utils.IoBind((*inConn), outConn, func(isSrcErr bool, err error) {
		log.Printf("conn %s - %s - %s -%s released [%s]", inAddr, inLocalAddr, outLocalAddr, outAddr, address)
		utils.CloseConn(inConn)
		utils.CloseConn(&outConn)
	}, func(n int, d bool) {}, 0)
	log.Printf("conn %s - %s - %s - %s connected [%s]", inAddr, inLocalAddr, outLocalAddr, outAddr, address)
	return
}

func (s *SOCKS) InitOutConnPool() {
	if *s.cfg.ParentType == TYPE_TLS || *s.cfg.ParentType == TYPE_TCP {
		s.outPool = utils.NewOutPool(
			*s.cfg.CheckParentInterval,
			*s.cfg.ParentType == TYPE_TLS,
			s.cfg.CertBytes, s.cfg.KeyBytes,
			*s.cfg.Parent,
			*s.cfg.Timeout,
			*s.cfg.PoolSize,
			*s.cfg.PoolSize*2,
		)
	}
}

func (s *SOCKS) InitBasicAuth() (err error) {
	s.basicAuth = utils.NewBasicAuth()
	if *s.cfg.AuthFile != "" {
		var n = 0
		n, err = s.basicAuth.AddFromFile(*s.cfg.AuthFile)
		if err != nil {
			err = fmt.Errorf("auth-file ERR:%s", err)
			return
		}
		log.Printf("auth data added from file %d , total:%d", n, s.basicAuth.Total())
	}
	if len(*s.cfg.Auth) > 0 {
		n := s.basicAuth.Add(*s.cfg.Auth)
		log.Printf("auth data added %d, total:%d", n, s.basicAuth.Total())
	}
	return
}

func (s *SOCKS) IsBasicAuth() bool {
	return *s.cfg.AuthFile != "" || len(*s.cfg.Auth) > 0
}
