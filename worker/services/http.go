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

	"github.com/google/uuid"
	"github.com/snail007/goproxy/manager"
	"github.com/snail007/goproxy/utils"
)

type HTTP struct {
	outPool   utils.OutPool
	cfg       HTTPArgs
	checker   utils.Checker
	basicAuth utils.BasicAuth
	worker    *manager.Worker
}

func NewHTTP() Service {
	return &HTTP{
		outPool:   utils.OutPool{},
		cfg:       HTTPArgs{},
		checker:   utils.Checker{},
		basicAuth: utils.BasicAuth{},
	}
}
func (s *HTTP) InitService() {
	s.InitBasicAuth()
	if s.worker.UpstreamManager == nil || !s.worker.UpstreamManager.HasUpstreams() {
		s.checker = utils.NewChecker(*s.cfg.HTTPTimeout, int64(*s.cfg.Interval), *s.cfg.Blocked, *s.cfg.Direct)
	}
}
func (s *HTTP) StopService() {
	if s.outPool.Pool != nil {
		s.outPool.Pool.ReleaseAll()
	}
}
func (s *HTTP) Start(args interface{}, worker *manager.Worker) (err error) {
	s.cfg = args.(HTTPArgs)
	s.worker = worker

	//add connection pool for upstream connections later
	/*if *s.cfg.Parent != "" {
		log.Printf("use %s parent %s", *s.cfg.ParentType, *s.cfg.Parent)
		s.InitOutConnPool()
	}*/

	s.InitService()
	s.basicAuth.Validator = worker.VerifyUser

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
	req, err := utils.NewHTTPRequest(&inConn, 4096, s.IsBasicAuth(), &s.basicAuth)
	if err != nil {
		if err != io.EOF {
			log.Printf("decoder error , form %s, ERR:%s", inConn.RemoteAddr(), err)
		}
		utils.CloseConn(&inConn)
		return
	}
	address := req.Host

	// Determine if we should use upstream proxy
	useProxy := false
	if s.worker.UpstreamManager != nil && s.worker.UpstreamManager.HasUpstreams() {
		useProxy = true
	} else if *s.cfg.Always {
		useProxy = true
	} else {
		if req.IsHTTPS() {
			s.checker.Add(address, true, req.Method, "", nil)
		} else {
			s.checker.Add(address, false, req.Method, req.URL, req.HeadBuf)
		}
		useProxy, _, _ = s.checker.IsBlocked(req.Host)
	}

	log.Printf("use proxy : %v, %s", useProxy, address)
	err = s.OutToTCP(useProxy, address, &inConn, &req)
	if err != nil {
		if s.worker.UpstreamManager != nil && s.worker.UpstreamManager.HasUpstreams() {
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
	var upstreamUser, upstreamPass string
	var currentUpstream *manager.Upstream

	if useProxy {
		// Try to get upstream from manager (round-robin)
		if s.worker.UpstreamManager != nil && s.worker.UpstreamManager.HasUpstreams() {
			upstream := s.worker.UpstreamManager.Next()
			if upstream != nil {
				currentUpstream = upstream
				upstreamAddr := upstream.GetAddress()
				upstreamUser = upstream.UpstreamUsername
				upstreamPass = upstream.UpstreamPassword
				log.Printf("[Upstream] Connecting to: %s (tag: %s)", upstreamAddr, upstream.UpstreamTag)

				// Measure connection latency for upstream health tracking
				connectStart := time.Now()
				outConn, err = utils.ConnectHost(upstreamAddr, *s.cfg.Timeout)
				connectLatency := time.Since(connectStart)

				// Record upstream latency in health collector
				if s.worker != nil && s.worker.HealthCollector != nil {
					s.worker.HealthCollector.RecordUpstreamLatency(
						upstream.UpstreamID,
						upstream.UpstreamTag,
						connectLatency,
						err != nil,
					)
				}
			} else {
				err = fmt.Errorf("no upstream available")
			}
		} else {
			err = fmt.Errorf("no upstream configured")
		}
	} else {
		outConn, err = utils.ConnectHost(address, *s.cfg.Timeout)
	}

	// Store current upstream for error tracking on connection close
	_ = currentUpstream // Will be used for per-request upstream error tracking if needed

	if err != nil {
		log.Printf("connect to %s , err:%s", "", err)
		utils.CloseConn(inConn)
		return
	}

	outAddr := outConn.RemoteAddr().String()
	outLocalAddr := outConn.LocalAddr().String()

	if req.IsHTTPS() && !useProxy {
		req.HTTPSReply()
	} else {
		httpReq, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(req.HeadBuf)))
		if err != nil {
			utils.CloseConn(&outConn)
			return err
		}
		httpReq.Header.Del("Proxy-Authorization")

		// Use upstream credentials if available
		if upstreamUser != "" && upstreamPass != "" {
			token := base64.StdEncoding.EncodeToString([]byte(upstreamUser + ":" + upstreamPass))
			httpReq.Header.Set("Proxy-Authorization", "Basic "+token)
			log.Printf("[Upstream] Using credentials for user: %s", upstreamUser)
		}

		var buf bytes.Buffer
		httpReq.WriteProxy(&buf)
		outConn.Write(buf.Bytes())
	}

	// Data usage tracking
	var bytesSent uint64
	var bytesReceived uint64
	username := req.GetBasicAuthUser()
	sourceIP := strings.Split(inAddr, ":")[0]

	// Parse destination host and port
	destHost, destPortStr, _ := net.SplitHostPort(req.Host)
	if destHost == "" {
		destHost = req.Host
	}
	var destPort uint16 = 80
	if req.IsHTTPS() {
		destPort = 443
	}
	if p, err := strconv.Atoi(destPortStr); err == nil {
		destPort = uint16(p)
	}

	// Track connection in HealthCollector
	if s.worker != nil && s.worker.HealthCollector != nil {
		s.worker.HealthCollector.IncrementConnection()
	}

	utils.IoBind((*inConn), outConn, func(isSrcErr bool, err error) {
		log.Printf("conn %s - %s - %s -%s released [%s]", inAddr, inLocalAddr, outLocalAddr, outAddr, req.Host)

		// Decrement connection count
		if s.worker != nil && s.worker.HealthCollector != nil {
			s.worker.HealthCollector.DecrementConnection()
			// Record success/error
			if err != nil {
				s.worker.HealthCollector.RecordError()
			} else {
				s.worker.HealthCollector.RecordSuccess()
			}
		}

		// Send data usage to Captain when connection closes
		if s.worker != nil && (bytesSent > 0 || bytesReceived > 0) {
			poolID, poolName := s.worker.GetPoolInfo()
			workerUUID, _ := uuid.Parse(s.worker.WorkerID)
			poolUUID, _ := uuid.Parse(poolID)

			usage := manager.UserDataUsage{
				UserID:          uuid.Nil,
				Username:        username,
				PoolID:          poolUUID,
				PoolName:        poolName,
				WorkerID:        workerUUID,
				WorkerRegion:    s.worker.Pool.Region,
				BytesSent:       atomic.LoadUint64(&bytesSent),
				BytesReceived:   atomic.LoadUint64(&bytesReceived),
				SourceIP:        sourceIP,
				Protocol:        "HTTP",
				DestinationHost: destHost,
				DestinationPort: destPort,
				StatusCode:      200, // Default success
			}
			if req.IsHTTPS() {
				usage.Protocol = "HTTPS"
			}

			s.worker.SendDataUsage(usage)
		}

		utils.CloseConn(inConn)
		utils.CloseConn(&outConn)
	}, func(n int, isDownload bool) {
		// Track bytes transferred
		if isDownload {
			atomic.AddUint64(&bytesReceived, uint64(n))
		} else {
			atomic.AddUint64(&bytesSent, uint64(n))
		}
		// Track throughput in HealthCollector
		if s.worker != nil && s.worker.HealthCollector != nil {
			s.worker.HealthCollector.AddThroughput(uint64(n))
		}
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
			//address for the connection pool
			"",
			*s.cfg.Timeout,
			*s.cfg.PoolSize,
			*s.cfg.PoolSize*2,
		)
	}
}
func (s *HTTP) InitBasicAuth() (err error) {
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
func (s *HTTP) IsBasicAuth() bool {
	return *s.cfg.AuthFile != "" || len(*s.cfg.Auth) > 0
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
