package services

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/snail007/goproxy/manager"
	"github.com/snail007/goproxy/utils"
)

type HTTP struct {
	outPool utils.OutPool
	cfg     HTTPArgs
	checker utils.Checker
	worker  *manager.WorkerManager
}

func NewHTTP() Service {
	return &HTTP{
		outPool: utils.OutPool{},
		cfg:     HTTPArgs{},
		checker: utils.Checker{},
	}
}

func (s *HTTP) InitService() {
	time.Sleep(time.Second * 5)
	if s.worker.HasUpstreams() {
		s.InitOutConnPool()
		s.checker = utils.NewChecker(*s.cfg.HTTPTimeout, int64(*s.cfg.Interval), *s.cfg.Blocked, *s.cfg.Direct)
	}
}

func (s *HTTP) StopService() {
	if s.outPool.UpstreamPool != nil {
		for _, pool := range s.outPool.UpstreamPool {
			(*pool).ReleaseAll()
		}
	}
}

func (s *HTTP) Start(args interface{}, worker *manager.WorkerManager) (err error) {
	s.cfg = args.(HTTPArgs)
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
	log.Printf("%s http(s) proxy on %s", *s.cfg.LocalType, (*sc.Listener).Addr())
	return
}

func (s *HTTP) Clean() {
	s.StopService()
}

func (s *HTTP) callback(inConn net.Conn) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("http(s) conn handler crashed with err : %s \nstack: %s", err, string(debug.Stack()))
		}
	}()
	req, err := utils.NewHTTPRequest(&inConn, 4096, s.worker.VerifyUser)
	if err != nil {
		if err != io.EOF {
			log.Printf("decoder error , form %s, ERR:%s", inConn.RemoteAddr(), err)
		}
		utils.CloseConn(&inConn)
		return
	}
	address := req.Host

	if err := s.worker.AddUserConnection(req.User); err != nil {
		log.Printf("add user connection failed, err: %s", err)
		inConn.Write([]byte("HTTP/1.1 429 Too Many Requests\r\n\r\n"))
		utils.CloseConn(&inConn)
		return
	}

	// Determine if we should use upstream proxy
	useProxy := false
	if s.worker.HasUpstreams() {
		if *s.cfg.Always {
			useProxy = true
		} else {
			if req.IsHTTPS() {
				s.checker.Add(address, true, req.Method, "", nil)
			} else {
				s.checker.Add(address, false, req.Method, req.URL, req.HeadBuf)
			}
			useProxy, _, _ = s.checker.IsBlocked(req.Host)
		}
	}

	log.Printf("use proxy : %v, %s", useProxy, address)
	err = s.OutToTCP(useProxy, address, &inConn, &req)
	if err != nil {
		if s.worker.HasUpstreams() {
			log.Printf("connect to %s parent %s fail", *s.cfg.ParentType, "")
		} else {
			log.Printf("connect to %s fail, ERR:%s", address, err)
		}
		utils.CloseConn(&inConn)
	}
}

func (s *HTTP) OutToTCP(useProxy bool, address string, inConn *net.Conn, req *utils.HTTPRequest) (err error) {
	inAddr := (*inConn).RemoteAddr().String()
	inLocalAddr := (*inConn).LocalAddr().String()

	if s.IsDeadLoop(inLocalAddr, req.Host) {
		utils.CloseConn(inConn)
		err = fmt.Errorf("dead loop detected , %s", req.Host)
		return
	}

	var outConn net.Conn
	var upstream *manager.Upstream

	if useProxy {
		if s.worker.HasUpstreams() {
			upstream = s.worker.NextUpstream(req.User, req.Tag.Session)
			if upstream != nil {
				log.Printf("[Upstream] Connecting to: %s (tag: %s)", upstream.GetAddress(), upstream.UpstreamTag)
				connectStart := time.Now()
				out, err := s.outPool.GetConnFromConnectionPool(upstream.GetAddress())
				if err != nil {
					outConn, err = utils.ConnectHost(upstream.GetAddress(), *s.cfg.Timeout)
				} else {
					log.Println("[Upstream] Using connection from pool")
					outConn = out.(net.Conn)
				}
				connectLatency := time.Since(connectStart)
				s.worker.RecordUpstreamLatency(upstream, connectLatency, nil)
			} else {
				outConn, err = utils.ConnectHost(address, *s.cfg.Timeout)
				err = fmt.Errorf("no upstream available")
			}
		} else {
			err = fmt.Errorf("no upstream configured")
		}
	} else {
		outConn, err = utils.ConnectHost(address, *s.cfg.Timeout)
	}

	if err != nil {
		log.Printf("connect to %s , err:%s", "", err)
		utils.CloseConn(inConn)
		return
	}

	outAddr := outConn.RemoteAddr().String()
	outLocalAddr := outConn.LocalAddr().String()

	if req.IsHTTPS() && !useProxy {
		req.HTTPSReply()
	}
	if useProxy {
		err := connectUpstream(req, upstream, &outConn)
		if err != nil {
			return err
		}
	} else {
		outConn.Write(req.HeadBuf)
	}

	var bytesSent uint64
	var bytesReceived uint64
	sourceIP := strings.Split(inAddr, ":")[0]

	destHost, destPortStr, err := net.SplitHostPort(req.Host)
	if err != nil {
		destHost = req.Host
		destPortStr = "80"
	}
	var destPort uint16 = 80
	if req.IsHTTPS() {
		destPort = 443
	}
	if p, err := strconv.Atoi(destPortStr); err == nil {
		destPort = uint16(p)
	}

	s.worker.IncrementConnection()

	utils.IoBind((*inConn), outConn, func(isSrcErr bool, err error) {
		log.Printf("conn %s - %s - %s -%s released [%s]", inAddr, inLocalAddr, outLocalAddr, outAddr, req.Host)
		s.worker.DecrementConnection(err != nil)
		s.worker.RecordDataUsage(bytesSent, bytesReceived, req.User, sourceIP, destHost, destPort, req.IsHTTPS())
		s.worker.RemoveUserConnection(req.User)
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
	log.Printf("conn %s - %s - %s - %s connected [%s]", inAddr, inLocalAddr, outLocalAddr, outAddr, req.Host)
	return
}

func (s *HTTP) OutToUDP(inConn *net.Conn) (err error) {
	return
}

func (s *HTTP) InitOutConnPool() {
	if *s.cfg.ParentType == TYPE_TLS || *s.cfg.ParentType == TYPE_TCP {
		//dur int, isTLS bool, certBytes, keyBytes []byte,
		//parent string, timeout int, InitialCap int, MaxCap int
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

func (s *HTTP) IsDeadLoop(inLocalAddr string, host string) bool {
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

func connectUpstream(req *utils.HTTPRequest, upstream *manager.Upstream, outConn *net.Conn) error {
	httpReq, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(req.HeadBuf)))
	if err != nil {
		utils.CloseConn(outConn)
		return err
	}
	httpReq.Header.Del("Proxy-Authorization")
	if upstream.UpstreamUsername != "" && upstream.UpstreamPassword != "" {
		tag := convertTag(upstream.UpstreamUsername, upstream.UpstreamPassword, req.Tag, upstream.UpstreamProvider)
		log.Printf("[Upstream] Using tag: %s", tag)
		token := base64.StdEncoding.EncodeToString([]byte(tag))
		httpReq.Header.Set("Proxy-Authorization", "Basic "+token)
		log.Printf("[Upstream] Using credentials for user: %s", upstream.UpstreamUsername)
	}
	var buf bytes.Buffer
	httpReq.WriteProxy(&buf)
	(*outConn).Write(buf.Bytes())
	return nil
}

func convertTag(username, password string, tag utils.Tag, upstream string) string {
	newstring := ""
	switch upstream {
	case "netnut":
		newstring += username
		if tag.Country != "" && (tag.City != "" || tag.State != "") {
			newstring += "-res_sc-" + tag.Country
			if tag.State != "" {
				newstring += "_" + tag.State
			}
			if tag.City != "" {
				newstring += "_" + tag.City
			}
		} else {
			newstring += "-res-" + tag.Country
		}
		if tag.Session != "" {
			newstring += "-sid-" + tag.Session
		}
		newstring += ":" + password
	case "geonode":
		newstring += username
		if tag.Country != "" {
			newstring += "-country-" + tag.Country
		}
		/*if tag.State != "" {
			newstring += "_" + tag.State
		}*/
		if tag.City != "" {
			newstring += "-city-" + tag.City
		}
		if tag.Session != "" {
			newstring += "-session-" + tag.Session
		}
		if tag.Lifetime > 0 {
			newstring += "-lifetime-" + strconv.Itoa(tag.Lifetime)
		}
		newstring += ":" + password
	case "iproyal":
		newstring += username + ":" + password
		if tag.Country != "" {
			newstring += "_country-" + tag.Country
		}
		/*if tag.State != "" {
			newstring += "_" + tag.State
		}*/
		if tag.City != "" {
			newstring += "_city-" + tag.City
		}
		if tag.Session != "" {
			newstring += "_session-" + tag.Session
		}
		if tag.Lifetime > 0 {
			newstring += "_lifetime-" + strconv.Itoa(tag.Lifetime) + "m"
		}
	}
	return newstring
}
