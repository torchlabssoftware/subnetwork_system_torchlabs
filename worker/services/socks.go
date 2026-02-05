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
	"sync"
	"sync/atomic"
	"time"

	"github.com/snail007/goproxy/manager"
	"github.com/snail007/goproxy/utils"
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

	if err := s.worker.AddUserConnection(user); err != nil {
		log.Printf("add user connection failed, err: %s", err)
		s.sendReply(&inConn, SOCKS5_REP_CONN_NOT_ALLOWED)
		utils.CloseConn(&inConn)
		return
	}

	address, cmd, err := s.handleRequest(&inConn)
	if err != nil {
		log.Printf("socks5 request error from %s: %s", inConn.RemoteAddr(), err)
		s.worker.RemoveUserConnection(user)
		utils.CloseConn(&inConn)
		return
	}

	// Determine if we should use upstream proxy
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

	if cmd == SOCKS5_CMD_UDP {
		err = s.handleUDP(&inConn, address, user)
		if err != nil {
			log.Printf("socks5 udp error from %s: %s", inConn.RemoteAddr(), err)
		}
		s.worker.RemoveUserConnection(user)
		utils.CloseConn(&inConn)
		return
	} else {
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
		return
	}

}

func (s *SOCKS) handleUDP(inConn *net.Conn, clientAddr string, user string) error {
	udpAddr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		return fmt.Errorf("failed to resolve udp addr: %w", err)
	}
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("failed to listen udp: %w", err)
	}
	defer udpConn.Close()

	localAddr := (*inConn).LocalAddr().(*net.TCPAddr)
	bindPort := udpConn.LocalAddr().(*net.UDPAddr).Port

	// Create reply
	reply := []byte{SOCKS5_VERSION, SOCKS5_REP_SUCCESS, 0x00, SOCKS5_ATYP_IPV4}
	reply = append(reply, localAddr.IP.To4()...)
	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, uint16(bindPort))
	reply = append(reply, portBytes...)

	if _, err := (*inConn).Write(reply); err != nil {
		return fmt.Errorf("failed to write reply: %w", err)
	}

	// Keep TCP open and wait for it to close
	closeUDP := make(chan struct{})
	go func() {
		io.Copy(io.Discard, *inConn)
		close(closeUDP)
	}()

	sessions := make(map[string]net.Conn)
	var sessionsMu sync.Mutex
	defer func() {
		sessionsMu.Lock()
		for _, conn := range sessions {
			conn.Close()
		}
		sessionsMu.Unlock()
	}()

	buf := make([]byte, 65535)
	for {
		// Check if TCP connection is still alive
		select {
		case <-closeUDP:
			return nil
		default:
		}

		udpConn.SetReadDeadline(time.Now().Add(time.Second))
		n, cAddr, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			return nil // Connection closed or other error
		}

		if n < 10 {
			continue
		}

		// Validate RSV
		if buf[0] != 0x00 || buf[1] != 0x00 {
			continue
		}

		frag := buf[2]
		if frag != 0 {
			continue // Fragmentation not supported
		}

		var targetHost string
		var targetPort int
		var headerLen int

		atyp := buf[3]
		switch atyp {
		case SOCKS5_ATYP_IPV4:
			targetHost = net.IP(buf[4:8]).String()
			targetPort = int(binary.BigEndian.Uint16(buf[8:10]))
			headerLen = 10
		case SOCKS5_ATYP_DOMAIN:
			domainLen := int(buf[4])
			targetHost = string(buf[5 : 5+domainLen])
			targetPort = int(binary.BigEndian.Uint16(buf[5+domainLen : 5+domainLen+2]))
			headerLen = 5 + domainLen + 2
		case SOCKS5_ATYP_IPV6:
			targetHost = net.IP(buf[4:20]).String()
			targetPort = int(binary.BigEndian.Uint16(buf[20:22]))
			headerLen = 22
		default:
			continue
		}

		targetAddrStr := fmt.Sprintf("%s:%d", targetHost, targetPort)
		payload := make([]byte, n-headerLen)
		copy(payload, buf[headerLen:n])

		sessionsMu.Lock()
		conn, ok := sessions[targetAddrStr]
		if !ok {
			// Create new session
			tAddr, err := net.ResolveUDPAddr("udp", targetAddrStr)
			if err != nil {
				sessionsMu.Unlock()
				continue
			}

			newConn, err := net.DialUDP("udp", nil, tAddr)
			if err != nil {
				sessionsMu.Unlock()
				continue
			}
			conn = newConn
			sessions[targetAddrStr] = conn

			// Start response relay for this session
			go func(targetConn net.Conn, clientUDPAddr *net.UDPAddr, tHost string, tPort int) {
				respBuf := make([]byte, 65535)
				for {
					targetConn.SetReadDeadline(time.Now().Add(time.Minute * 2))
					rn, err := targetConn.Read(respBuf)
					if err != nil {
						sessionsMu.Lock()
						delete(sessions, targetAddrStr)
						sessionsMu.Unlock()
						targetConn.Close()
						return
					}

					// Build SOCKS5 UDP header for response
					respHeader := make([]byte, 0, 22)
					respHeader = append(respHeader, 0, 0, 0) // RSV, FRAG

					// Use original target address type/info
					if ip4 := net.ParseIP(tHost).To4(); ip4 != nil {
						respHeader = append(respHeader, SOCKS5_ATYP_IPV4)
						respHeader = append(respHeader, ip4...)
					} else if ip6 := net.ParseIP(tHost).To16(); ip6 != nil {
						respHeader = append(respHeader, SOCKS5_ATYP_IPV6)
						respHeader = append(respHeader, ip6...)
					} else {
						respHeader = append(respHeader, SOCKS5_ATYP_DOMAIN)
						respHeader = append(respHeader, byte(len(tHost)))
						respHeader = append(respHeader, []byte(tHost)...)
					}

					pBytes := make([]byte, 2)
					binary.BigEndian.PutUint16(pBytes, uint16(tPort))
					respHeader = append(respHeader, pBytes...)

					finalPacket := append(respHeader, respBuf[:rn]...)
					udpConn.WriteToUDP(finalPacket, clientUDPAddr)

					// Telemetry
					s.worker.RecordDataUsage(0, uint64(rn), user, clientUDPAddr.IP.String(), tHost, uint16(tPort), false)
					s.worker.AddThroughput(uint64(rn))
				}
			}(conn, cAddr, targetHost, targetPort)
		}
		sessionsMu.Unlock()

		// Send to target
		_, err = conn.Write(payload)
		if err == nil {
			s.worker.RecordDataUsage(uint64(len(payload)), 0, user, cAddr.IP.String(), targetHost, uint16(targetPort), false)
			s.worker.AddThroughput(uint64(len(payload)))
		}
	}
	return nil
}

func (s *SOCKS) handleHandshake(inConn *net.Conn) (string, utils.Tag, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(*inConn, header); err != nil {
		return "", utils.Tag{}, fmt.Errorf("failed to read header: %w", err)
	}

	if header[0] != SOCKS5_VERSION {
		return "", utils.Tag{}, fmt.Errorf("unsupported SOCKS version: %d", header[0])
	}

	numMethods := int(header[1])
	methods := make([]byte, numMethods)
	if _, err := io.ReadFull(*inConn, methods); err != nil {
		return "", utils.Tag{}, fmt.Errorf("failed to read auth methods: %w", err)
	}

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

	(*inConn).Write([]byte{SOCKS5_VERSION, SOCKS5_AUTH_PASSWORD})

	user, tag, err := s.handlePasswordAuth(inConn)
	if err != nil {
		return "", utils.Tag{}, err
	}

	return user, tag, nil
}

func (s *SOCKS) handlePasswordAuth(inConn *net.Conn) (string, utils.Tag, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(*inConn, header); err != nil {
		return "", utils.Tag{}, fmt.Errorf("failed to read auth header: %w", err)
	}

	if header[0] != 0x01 {
		return "", utils.Tag{}, fmt.Errorf("unsupported auth version: %d", header[0])
	}

	usernameLen := int(header[1])
	username := make([]byte, usernameLen)
	if _, err := io.ReadFull(*inConn, username); err != nil {
		return "", utils.Tag{}, fmt.Errorf("failed to read username: %w", err)
	}

	passLenByte := make([]byte, 1)
	if _, err := io.ReadFull(*inConn, passLenByte); err != nil {
		return "", utils.Tag{}, fmt.Errorf("failed to read password length: %w", err)
	}

	passwordLen := int(passLenByte[0])
	password := make([]byte, passwordLen)
	if _, err := io.ReadFull(*inConn, password); err != nil {
		return "", utils.Tag{}, fmt.Errorf("failed to read password: %w", err)
	}

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

	actualPassword := tagArray[0]
	if !s.worker.VerifyUser(string(username), actualPassword) {
		(*inConn).Write([]byte{0x01, 0x01})
		return "", utils.Tag{}, fmt.Errorf("authentication failed for user: %s", string(username))
	}

	log.Printf("socks5 auth success for user: %s", string(username))
	(*inConn).Write([]byte{0x01, 0x00})
	return string(username), tag, nil
}

func (s *SOCKS) handleRequest(inConn *net.Conn) (string, byte, error) {
	header := make([]byte, 4)
	if _, err := io.ReadFull(*inConn, header); err != nil {
		return "", 0, fmt.Errorf("failed to read request header: %w", err)
	}

	if header[0] != SOCKS5_VERSION {
		return "", 0, fmt.Errorf("unsupported SOCKS version: %d", header[0])
	}

	cmd := header[1]
	atyp := header[3]

	if cmd != SOCKS5_CMD_CONNECT && cmd != SOCKS5_CMD_UDP {
		s.sendReply(inConn, SOCKS5_REP_CMD_NOT_SUPPORTED)
		return "", 0, fmt.Errorf("unsupported command: %d", cmd)
	}

	var host string
	switch atyp {
	case SOCKS5_ATYP_IPV4:
		addr := make([]byte, 4)
		if _, err := io.ReadFull(*inConn, addr); err != nil {
			return "", 0, fmt.Errorf("failed to read IPv4 address: %w", err)
		}
		host = net.IP(addr).String()

	case SOCKS5_ATYP_DOMAIN:
		lenByte := make([]byte, 1)
		if _, err := io.ReadFull(*inConn, lenByte); err != nil {
			return "", 0, fmt.Errorf("failed to read domain length: %w", err)
		}
		domainLen := int(lenByte[0])
		domain := make([]byte, domainLen)
		if _, err := io.ReadFull(*inConn, domain); err != nil {
			return "", 0, fmt.Errorf("failed to read domain: %w", err)
		}
		host = string(domain)

	case SOCKS5_ATYP_IPV6:
		addr := make([]byte, 16)
		if _, err := io.ReadFull(*inConn, addr); err != nil {
			return "", 0, fmt.Errorf("failed to read IPv6 address: %w", err)
		}
		host = net.IP(addr).String()

	default:
		s.sendReply(inConn, SOCKS5_REP_ATYP_NOT_SUPPORTED)
		return "", 0, fmt.Errorf("unsupported address type: %d", atyp)
	}

	portBytes := make([]byte, 2)
	if _, err := io.ReadFull(*inConn, portBytes); err != nil {
		return "", 0, fmt.Errorf("failed to read port: %w", err)
	}
	port := binary.BigEndian.Uint16(portBytes)

	address := fmt.Sprintf("%s:%d", host, port)
	log.Printf("SOCKS5 Request: %s (CMD: %d)", address, cmd)

	return address, cmd, nil
}

func (s *SOCKS) sendReply(inConn *net.Conn, rep byte) {
	reply := []byte{SOCKS5_VERSION, rep, 0x00, SOCKS5_ATYP_IPV4, 0, 0, 0, 0, 0, 0}
	(*inConn).Write(reply)
}

func (s *SOCKS) OutToTCP(useProxy bool, address string, inConn *net.Conn, user string, tag utils.Tag) (err error) {
	inAddr := (*inConn).RemoteAddr().String()
	inLocalAddr := (*inConn).LocalAddr().String()

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
		return err
	}

	s.sendReply(inConn, SOCKS5_REP_SUCCESS)

	outAddr := outConn.RemoteAddr().String()
	outLocalAddr := outConn.LocalAddr().String()

	if useProxy {
		err = connectUpstreamSocks(tag, upstream, &outConn, address)
		if err != nil {
			utils.CloseConn(inConn)
			utils.CloseConn(&outConn)
			return err
		}
	}

	var bytesSent uint64
	var bytesReceived uint64
	sourceIP := strings.Split(inAddr, ":")[0]

	destHost, destPortStr, parseErr := net.SplitHostPort(address)
	if parseErr != nil {
		destHost = address
		destPortStr = "0"
	}
	var destPort uint16 = 0
	if p, convErr := strconv.Atoi(destPortStr); convErr == nil {
		destPort = uint16(p)
	}
	s.worker.IncrementConnection()

	utils.IoBind((*inConn), outConn, func(isSrcErr bool, err error) {
		log.Printf("conn %s - %s - %s -%s released [%s]", inAddr, inLocalAddr, outLocalAddr, outAddr, address)
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
