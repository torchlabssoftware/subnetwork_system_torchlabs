package services

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"runtime/debug"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/snail007/goproxy/manager"
	"github.com/snail007/goproxy/utils"
)

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
	outPool utils.OutPool
	cfg     SOCKSArgs
	checker utils.Checker
	worker  *manager.WorkerManager
}

func NewSOCKS() Service {
	return &SOCKS{
		outPool: utils.OutPool{},
		cfg:     SOCKSArgs{},
		checker: utils.Checker{},
	}
}

func (s *SOCKS) InitService() {
	time.Sleep(time.Second * 5)
	if s.worker.HasUpstreams() {
		s.InitOutConnPool()
		s.checker = utils.NewChecker(*s.cfg.HTTPTimeout, int64(*s.cfg.Interval), *s.cfg.Blocked, *s.cfg.Direct)
	}
}

func (s *SOCKS) StopService() {
	if s.outPool.UpstreamPool != nil {
		for _, pool := range s.outPool.UpstreamPool {
			(*pool).ReleaseAll()
		}
	}
}

func (s *SOCKS) Start(args interface{}, worker *manager.WorkerManager) (err error) {
	s.cfg = args.(SOCKSArgs)
	s.worker = worker

	/*if *s.cfg.Parent != "" {
		log.Printf("use %s parent %s", *s.cfg.ParentType, *s.cfg.Parent)
		s.InitOutConnPool()
	}*/

	s.InitService()

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

	user, tag, err := s.handleHandshake(&inConn)
	if err != nil {
		log.Printf("socks5 handshake error from %s: %s", inConn.RemoteAddr(), err)
		utils.CloseConn(&inConn)
		return
	}

	// Check user connection limits
	if err := s.worker.AddUserConnection(user); err != nil {
		log.Printf("add user connection failed, err: %s", err)
		s.sendReply(&inConn, SOCKS5_REP_CONN_NOT_ALLOWED)
		utils.CloseConn(&inConn)
		return
	}

	// Handle SOCKS5 request
	address, err := s.handleRequest(&inConn)
	if err != nil {
		log.Printf("socks5 request error from %s: %s", inConn.RemoteAddr(), err)
		s.worker.RemoveUserConnection(user)
		utils.CloseConn(&inConn)
		return
	}

	// Determine if we should use upstream proxy (FIXED: logic was inverted)
	useProxy := false
	if s.worker.HasUpstreams() {
		if *s.cfg.Always {
			useProxy = true
		} else {
			s.checker.Add(address, true, "CONNECT", "", nil)
			useProxy, _, _ = s.checker.IsBlocked(address)
		}
	}
	log.Printf("use proxy : %v, %s", useProxy, address)

	err = s.OutToTCP(useProxy, address, &inConn, user, tag)
	if err != nil {
		if s.worker.HasUpstreams() {
			log.Printf("connect to %s parent %s fail", *s.cfg.ParentType, "")
		} else {
			log.Printf("connect to %s fail, ERR:%s", address, err)
		}
		s.worker.RemoveUserConnection(user)
		utils.CloseConn(&inConn)
	}
}

func (s *SOCKS) handleHandshake(inConn *net.Conn) (string, utils.Tag, error) {
	// Read version and number of auth methods
	header := make([]byte, 2)
	if _, err := io.ReadFull(*inConn, header); err != nil {
		return "", utils.Tag{}, fmt.Errorf("failed to read header: %w", err)
	}

	if header[0] != SOCKS5_VERSION {
		return "", utils.Tag{}, fmt.Errorf("unsupported SOCKS version: %d", header[0])
	}

	// Read auth methods
	numMethods := int(header[1])
	methods := make([]byte, numMethods)
	if _, err := io.ReadFull(*inConn, methods); err != nil {
		return "", utils.Tag{}, fmt.Errorf("failed to read auth methods: %w", err)
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
		return "", utils.Tag{}, fmt.Errorf("client doesn't support password auth")
	}

	// Request password auth
	(*inConn).Write([]byte{SOCKS5_VERSION, SOCKS5_AUTH_PASSWORD})

	// Handle password auth
	user, tag, err := s.handlePasswordAuth(inConn)
	if err != nil {
		return "", utils.Tag{}, err
	}

	return user, tag, nil
}

func (s *SOCKS) handlePasswordAuth(inConn *net.Conn) (string, utils.Tag, error) {
	// Read auth version
	header := make([]byte, 2)
	if _, err := io.ReadFull(*inConn, header); err != nil {
		return "", utils.Tag{}, fmt.Errorf("failed to read auth header: %w", err)
	}

	// auth version should be 0x01
	if header[0] != 0x01 {
		return "", utils.Tag{}, fmt.Errorf("unsupported auth version: %d", header[0])
	}

	// Read username
	usernameLen := int(header[1])
	username := make([]byte, usernameLen)
	if _, err := io.ReadFull(*inConn, username); err != nil {
		return "", utils.Tag{}, fmt.Errorf("failed to read username: %w", err)
	}

	// Read password length
	passLenByte := make([]byte, 1)
	if _, err := io.ReadFull(*inConn, passLenByte); err != nil {
		return "", utils.Tag{}, fmt.Errorf("failed to read password length: %w", err)
	}

	// Read password
	passwordLen := int(passLenByte[0])
	password := make([]byte, passwordLen)
	if _, err := io.ReadFull(*inConn, password); err != nil {
		return "", utils.Tag{}, fmt.Errorf("failed to read password: %w", err)
	}

	// Parse routing tags from password (format: password-country-US-session-abc123)
	passwordStr := string(password)
	tagArray := strings.Split(passwordStr, "-")
	tag := utils.Tag{}
	for i, v := range tagArray {
		if v == "country" && i+1 < len(tagArray) {
			tag.Country = tagArray[i+1]
		}
		if v == "state" && i+1 < len(tagArray) {
			tag.State = tagArray[i+1]
		}
		if v == "city" && i+1 < len(tagArray) {
			tag.City = tagArray[i+1]
		}
		if v == "session" && i+1 < len(tagArray) {
			tag.Session = tagArray[i+1]
		}
		if v == "lifetime" && i+1 < len(tagArray) {
			tag.Lifetime, _ = strconv.Atoi(tagArray[i+1])
		}
	}

	// Validate credentials (use first part of password before any tags)
	actualPassword := tagArray[0]
	if !s.worker.VerifyUser(string(username), actualPassword) {
		(*inConn).Write([]byte{0x01, 0x01}) // Auth failed
		return "", utils.Tag{}, fmt.Errorf("authentication failed for user: %s", string(username))
	}

	log.Printf("socks5 auth success for user: %s", string(username))
	(*inConn).Write([]byte{0x01, 0x00}) // Auth success
	return string(username), tag, nil
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

func (s *SOCKS) OutToTCP(useProxy bool, address string, inConn *net.Conn, user string, tag utils.Tag) (err error) {
	inAddr := (*inConn).RemoteAddr().String()
	inLocalAddr := (*inConn).LocalAddr().String()

	// Dead loop detection
	if s.IsDeadLoop(inLocalAddr, address) {
		utils.CloseConn(inConn)
		err = fmt.Errorf("dead loop detected , %s", address)
		return
	}

	var outConn net.Conn
	var upstream *manager.Upstream

	if useProxy {
		if s.worker.HasUpstreams() {
			upstream = s.worker.NextUpstream(user, tag.Session)
			if upstream != nil {
				log.Printf("[Upstream] Connecting to: %s (tag: %s)", upstream.GetAddress(), upstream.UpstreamTag)
				connectStart := time.Now()
				out, poolErr := s.outPool.GetConnFromConnectionPool(upstream.GetAddress())
				if poolErr != nil {
					outConn, err = utils.ConnectHost(upstream.GetAddress(), *s.cfg.Timeout)
				} else {
					log.Println("[Upstream] Using connection from pool")
					outConn = out.(net.Conn)
				}
				connectLatency := time.Since(connectStart)
				s.worker.RecordUpstreamLatency(upstream, connectLatency, err)
			} else {
				err = fmt.Errorf("no upstream available")
			}
		} else {
			err = fmt.Errorf("no upstream configured")
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

	// Data usage tracking
	var bytesSent uint64
	var bytesReceived uint64
	sourceIP := strings.Split(inAddr, ":")[0]

	// Parse destination host and port
	destHost, destPortStr, parseErr := net.SplitHostPort(address)
	if parseErr != nil {
		destHost = address
		destPortStr = "0"
	}
	var destPort uint16 = 0
	if p, convErr := strconv.Atoi(destPortStr); convErr == nil {
		destPort = uint16(p)
	}

	// Track connection
	s.worker.IncrementConnection()

	utils.IoBind((*inConn), outConn, func(isSrcErr bool, err error) {
		log.Printf("conn %s - %s - %s -%s released [%s]", inAddr, inLocalAddr, outLocalAddr, outAddr, address)

		// Decrement connection and record data usage
		s.worker.DecrementConnection(err != nil)
		s.worker.RecordDataUsage(bytesSent, bytesReceived, user, sourceIP, destHost, destPort, false)
		s.worker.RemoveUserConnection(user)

		utils.CloseConn(inConn)
		utils.CloseConn(&outConn)
	}, func(n int, isDownload bool) {
		if isDownload {
			atomic.AddUint64(&bytesReceived, uint64(n))
		} else {
			atomic.AddUint64(&bytesSent, uint64(n))
		}
		s.worker.AddThroughput(uint64(n))
	}, 0)
	log.Printf("conn %s - %s - %s - %s connected [%s]", inAddr, inLocalAddr, outLocalAddr, outAddr, address)
	return
}

func (s *SOCKS) InitOutConnPool() {
	if *s.cfg.ParentType == TYPE_TLS || *s.cfg.ParentType == TYPE_TCP {
		s.outPool = utils.NewOutPool(
			*s.cfg.CheckParentInterval,
			*s.cfg.ParentType == TYPE_TLS,
			s.cfg.CertBytes, s.cfg.KeyBytes,
			*s.cfg.Timeout,
			*s.cfg.PoolSize,
			*s.cfg.PoolSize*2,
			s.worker.GetUpstreamAddress(),
		)
	}
}

func (s *SOCKS) IsBasicAuth() bool {
	return *s.cfg.AuthFile != "" || len(*s.cfg.Auth) > 0
}

func (s *SOCKS) IsDeadLoop(inLocalAddr string, host string) bool {
	inIP, inPort, err := net.SplitHostPort(inLocalAddr)
	if err != nil {
		return false
	}
	outDomain, outPort, err := net.SplitHostPort(host)
	if err != nil {
		return false
	}
	if inPort == outPort {
		var outIPs []net.IP
		outIPs, err = net.LookupIP(outDomain)
		if err == nil {
			for _, ip := range outIPs {
				if ip.String() == inIP {
					return true
				}
			}
		}
		interfaceIPs, err := utils.GetAllInterfaceAddr()
		if err == nil {
			for _, localIP := range interfaceIPs {
				for _, outIP := range outIPs {
					if localIP.Equal(outIP) {
						return true
					}
				}
			}
		}
	}
	return false
}
