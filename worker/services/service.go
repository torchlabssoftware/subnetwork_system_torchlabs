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

	"github.com/snail007/goproxy/manager"
	"github.com/snail007/goproxy/utils"
)

type Service interface {
	Start(args interface{}, worker *manager.WorkerManager) (err error)
	Clean()
}

type ServiceItem struct {
	S    Service
	Args interface{}
	Name string
}

var servicesMap = map[string]*ServiceItem{}

// register the service item with properties
func Regist(name string, s Service, args interface{}) {
	servicesMap[name] = &ServiceItem{
		S:    s,
		Args: args,
		Name: name,
	}
}

// run the service in the arguments. do not try to run several services at the same time
func Run(name string, worker *manager.WorkerManager) (service *ServiceItem, err error) {
	service, ok := servicesMap[name]
	if ok {
		go func() {
			defer func() {
				err := recover()
				if err != nil {
					log.Fatalf("%s servcie crashed, ERR: %s\ntrace:%s", name, err, string(debug.Stack()))
				}
			}()
			err := service.S.Start(service.Args, worker)
			if err != nil {
				log.Fatalf("%s servcie fail, ERR: %s", name, err)
			}
		}()
	}
	if !ok {
		err = fmt.Errorf("service %s not found", name)
	}
	return
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

func connectUpstreamSocks(tag utils.Tag, upstream *manager.Upstream, outConn *net.Conn, address string) error {
	_, err := (*outConn).Write([]byte{SOCKS5_VERSION, 0x01, SOCKS5_AUTH_PASSWORD})
	if err != nil {
		log.Printf("[Upstream] Failed to send auth request: %s", err)
		return err
	}

	reply := make([]byte, 2)
	if _, err = io.ReadFull(*outConn, reply); err != nil {
		log.Printf("[Upstream] Failed to read auth reply: %s", err)
		return err
	}
	if reply[1] != 0x02 {
		return fmt.Errorf("upstream does not accept username/password auth")
	}

	newTag := strings.Split(convertTag(upstream.UpstreamUsername, upstream.UpstreamPassword, tag, upstream.UpstreamProvider), ":")
	username := newTag[0]
	password := newTag[1]

	authReq := []byte{0x01, byte(len(username))}
	authReq = append(authReq, []byte(username)...)
	authReq = append(authReq, byte(len(password)))
	authReq = append(authReq, []byte(password)...)

	if _, err = (*outConn).Write(authReq); err != nil {
		return err
	}

	authResp := make([]byte, 2)
	if _, err = io.ReadFull(*outConn, authResp); err != nil {
		return err
	}
	if authResp[1] != 0x00 {
		return fmt.Errorf("upstream authentication failed")
	}

	host, portStr, _ := net.SplitHostPort(address)
	port, _ := strconv.Atoi(portStr)

	req := []byte{
		0x05,
		0x01,
		0x00,
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return err
	}
	var ip4 net.IP
	for _, ip := range ips {
		if v4 := ip.To4(); v4 != nil {
			ip4 = v4
			break
		}
	}
	if ip4 == nil {
		return fmt.Errorf("no IPv4 address found")
	}

	req = append(req, 0x01) // IPv4
	req = append(req, ip4...)

	req = append(req, byte(port>>8), byte(port))

	if _, err = (*outConn).Write(req); err != nil {
		return err
	}

	resp := make([]byte, 4)
	if _, err = io.ReadFull(*outConn, resp); err != nil {
		return err
	}
	if resp[1] != 0x00 {
		return fmt.Errorf("upstream CONNECT failed, code=%d", resp[1])
	}

	switch resp[3] {
	case 0x01:
		io.ReadFull(*outConn, make([]byte, 6))
	case 0x03:
		l := make([]byte, 1)
		io.ReadFull(*outConn, l)
		io.ReadFull(*outConn, make([]byte, int(l[0])+2))
	case 0x04:
		io.ReadFull(*outConn, make([]byte, 18))
	}

	log.Printf("[Upstream] SOCKS CONNECT success to %s", address)
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
