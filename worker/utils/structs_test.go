package utils

import (
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
)

func TestChecker_NewChecker(t *testing.T) {
	timeout := 5000
	interval := int64(300)
	blockedFile := "blocked_test.txt"
	directFile := "direct_test.txt"
	os.WriteFile(blockedFile, []byte("blocked.com\n*.bad.com"), 0644)
	os.WriteFile(directFile, []byte("direct.com\ngoogle.com"), 0644)
	defer os.Remove(blockedFile)
	defer os.Remove(directFile)
	checker := NewChecker(timeout, interval, blockedFile, directFile)
	if checker.interval != interval {
		t.Errorf("Interval should be %d, got %d", interval, checker.interval)
	}
	if checker.timeout != timeout {
		t.Errorf("Timeout should be %d, got %d", timeout, checker.timeout)
	}
	if !checker.blockedMap.Has("blocked.com") {
		t.Error("Blocked map should have blocked.com")
	}
	if !checker.directMap.Has("direct.com") {
		t.Error("Direct map should have direct.com")
	}
}

func TestChecker_LoadMap(t *testing.T) {
	checker := Checker{}
	dataMap := checker.loadMap("")
	if dataMap.Count() != 0 {
		t.Error("Empty file should result in empty map")
	}
	tempFile := "test_load_map.txt"
	os.WriteFile(tempFile, []byte("domain1.com\ndomain2.com\r\n  domain3.com  "), 0644)
	defer os.Remove(tempFile)
	dataMap = checker.loadMap(tempFile)
	if dataMap.Count() != 3 {
		t.Errorf("Expected 3 domains, got %d", dataMap.Count())
	}
	if !dataMap.Has("domain1.com") || !dataMap.Has("domain2.com") || !dataMap.Has("domain3.com") {
		t.Error("Loaded map missing domains")
	}
}

func TestChecker_IsNeedCheck(t *testing.T) {
	checker := NewChecker(1000, 60, "", "")
	item := CheckerItem{Host: "example.com"}
	if !checker.isNeedCheck(item) {
		t.Error("New item should need check")
	}
	item = CheckerItem{Host: "example.com", SuccessCount: 5, FailCount: 2}
	if checker.isNeedCheck(item) {
		t.Error("Successful item should not need check")
	}
	item = CheckerItem{Host: "example.com", SuccessCount: 6, FailCount: 5}
	item = CheckerItem{Host: "example.com", SuccessCount: 6, FailCount: 5}
	if checker.isNeedCheck(item) {
		t.Error("Item with success > fail should not need check even if fail >= 5")
	}
	checker.directMap.Set("direct.com", true)
	item = CheckerItem{Host: "direct.com"}
	if checker.isNeedCheck(item) {
		t.Error("Item in direct map should not need check")
	}
	checker.blockedMap.Set("blocked.com", true)
	item = CheckerItem{Host: "blocked.com"}
	if checker.isNeedCheck(item) {
		t.Error("Item in blocked map should not need check")
	}
}

func TestChecker_IsBlocked(t *testing.T) {
	checker := NewChecker(1000, 60, "", "")
	checker.blockedMap.Set("bad.com", true)
	blocked, _, _ := checker.IsBlocked("bad.com")
	if !blocked {
		t.Error("bad.com should be blocked")
	}
	checker.directMap.Set("good.com", true)
	blocked, _, _ = checker.IsBlocked("good.com")
	if blocked {
		t.Error("good.com should not be blocked")
	}
	blocked, _, _ = checker.IsBlocked("unknown.com")
	if !blocked {
		t.Error("Unknown domain should be blocked by default")
	}
	checker.data.Set("test.com", CheckerItem{FailCount: 5, SuccessCount: 2})
	blocked, f, s := checker.IsBlocked("test.com")
	if !blocked || f != 5 || s != 2 {
		t.Errorf("test.com should be blocked (5 fail, 2 success), got %v, %d, %d", blocked, f, s)
	}
	checker.data.Set("test2.com", CheckerItem{FailCount: 2, SuccessCount: 5})
	blocked, _, _ = checker.IsBlocked("test2.com")
	if blocked {
		t.Error("test2.com should not be blocked (5 success, 2 fail)")
	}
}

func TestChecker_DomainIsInMap(t *testing.T) {
	checker := NewChecker(1000, 60, "", "")
	checker.blockedMap.Set("blocked.com", true)
	checker.directMap.Set("direct.net", true)
	tests := []struct {
		addr     string
		blocked  bool
		expected bool
	}{
		{"blocked.com", true, true},
		{"sub.blocked.com", true, true},
		{"direct.net", false, true},
		{"sub.direct.net", false, true},
		{"other.com", true, false},
		{"other.com", false, false},
	}
	for _, tt := range tests {
		result := checker.domainIsInMap(tt.addr, tt.blocked)
		if result != tt.expected {
			t.Errorf("domainIsInMap(%s, %v) = %v; want %v", tt.addr, tt.blocked, result, tt.expected)
		}
	}
}

func TestChecker_Add(t *testing.T) {
	checker := NewChecker(1000, 60, "", "")
	checker.Add("example.com:80", false, "GET", "http://example.com/", nil)
	if !checker.data.Has("example.com:80") {
		t.Error("Should have added example.com:80")
	}
	checker.Add("secure.com:443", true, "CONNECT", "secure.com:443", nil)
	if !checker.data.Has("secure.com:443") {
		t.Error("Should have added secure.com:443")
	}
	checker.Add("post.com:80", false, "POST", "http://post.com/", nil)
	if checker.data.Has("post.com:80") {
		t.Error("Should not add HTTP POST")
	}
	checker.directMap.Set("known.com", true)
	checker.Add("known.com:80", false, "GET", "http://known.com/", nil)
	if checker.data.Has("known.com:80") {
		t.Error("Should not add domain already in direct map")
	}
}

func TestHTTPRequest_Lifecycle(t *testing.T) {
	s, c := net.Pipe()
	defer s.Close()
	defer c.Close()
	validator := func(u, p string) bool {
		return u == "user" && p == "pass"
	}
	go func() {
		auth := base64.StdEncoding.EncodeToString([]byte("user:pass-country-US"))
		req := fmt.Sprintf("GET /index.html HTTP/1.1\r\nHost: example.com\r\nProxy-Authorization: Basic %s\r\n\r\n", auth)
		c.Write([]byte(req))
	}()
	req, err := NewHTTPRequest(&s, 1024, validator)
	if err != nil {
		t.Fatalf("NewHTTPRequest failed: %v", err)
	}
	if req.Method != "GET" {
		t.Errorf("Expected method GET, got %s", req.Method)
	}
	if req.Host != "example.com:80" {
		t.Errorf("Expected host example.com:80, got %s", req.Host)
	}
	if req.User != "user" {
		t.Errorf("Expected user user, got %s", req.User)
	}
	if req.Tag.Country != "US" {
		t.Errorf("Expected country US, got %s", req.Tag.Country)
	}
}

func TestHTTPRequest_HTTPS(t *testing.T) {
	s, c := net.Pipe()
	defer s.Close()
	defer c.Close()
	validator := func(u, p string) bool { return true }
	go func() {
		auth := base64.StdEncoding.EncodeToString([]byte("user:pass"))
		req := fmt.Sprintf("CONNECT example.com:443 HTTP/1.1\r\nProxy-Authorization: Basic %s\r\n\r\n", auth)
		c.Write([]byte(req))
	}()
	req, err := NewHTTPRequest(&s, 1024, validator)
	if err != nil {
		t.Fatalf("NewHTTPRequest failed: %v", err)
	}
	if !req.IsHTTPS() {
		t.Error("Should be HTTPS (CONNECT method)")
	}
	if req.Host != "example.com:443" {
		t.Errorf("Expected host example.com:443, got %s", req.Host)
	}
	go func() {
		buf := make([]byte, 1024)
		n, _ := c.Read(buf)
		if !strings.Contains(string(buf[:n]), "200 Connection established") {
			t.Errorf("Expected connection established reply, got %s", string(buf[:n]))
		}
	}()
	req.HTTPSReply()
}

func TestHTTPRequest_GetBasicAuthUser(t *testing.T) {
	auth := base64.StdEncoding.EncodeToString([]byte("testuser:secret"))
	req := HTTPRequest{
		HeadBuf: []byte(fmt.Sprintf("GET / HTTP/1.1\r\nProxy-Authorization: Basic %s\r\n\r\n", auth)),
	}
	user := req.GetBasicAuthUser()
	if user != "testuser" {
		t.Errorf("Expected testuser, got %s", user)
	}
	req.HeadBuf = []byte("GET / HTTP/1.1\r\n\r\n")
	if req.GetBasicAuthUser() != "" {
		t.Error("Expected empty user when header missing")
	}
}

func TestHTTPRequest_AddPortIfNot(t *testing.T) {
	req := HTTPRequest{Host: "example.com", Method: "GET"}
	req.addPortIfNot()
	if req.Host != "example.com:80" {
		t.Errorf("Expected example.com:80, got %s", req.Host)
	}
	req = HTTPRequest{Host: "example.com", Method: "CONNECT"}
	req.addPortIfNot()
	if req.Host != "example.com:443" {
		t.Errorf("Expected example.com:443, got %s", req.Host)
	}
	req = HTTPRequest{Host: "1.2.3.4", Method: "GET"}
	req.addPortIfNot()
	if req.Host != "1.2.3.4:80" {
		t.Errorf("Expected 1.2.3.4:80, got %s", req.Host)
	}
	req = HTTPRequest{Host: "[::1]", Method: "GET"}
	req.addPortIfNot()
	if req.Host != "[::1]:80" {
		t.Errorf("Expected [::1]:80, got %s", req.Host)
	}
	req = HTTPRequest{Host: "example.com:8080", Method: "GET"}
	req.addPortIfNot()
	if req.Host != "example.com:8080" {
		t.Errorf("Expected example.com:8080, got %s", req.Host)
	}
}

func TestConcurrentMap_Methods(t *testing.T) {
	cm := NewConcurrentMap()
	cm.Set("k1", "v1")
	if v, ok := cm.Get("k1"); !ok || v != "v1" {
		t.Error("Set/Get failed")
	}
	if !cm.Has("k1") || cm.Has("k2") {
		t.Error("Has failed")
	}
	if cm.IsEmpty() || cm.Count() != 1 {
		t.Error("IsEmpty/Count failed")
	}
	cm.Remove("k1")
	if cm.Has("k1") || !cm.IsEmpty() {
		t.Error("Remove failed")
	}
	cm.SetIfAbsent("k1", "v1")
	cm.SetIfAbsent("k1", "v2")
	if v, _ := cm.Get("k1"); v != "v1" {
		t.Error("SetIfAbsent failed")
	}
	cm.Set("k2", "v2")
	items := cm.Items()
	if len(items) != 2 {
		t.Error("Items failed")
	}
	count := 0
	for range cm.IterBuffered() {
		count++
	}
	if count != 2 {
		t.Error("IterBuffered failed")
	}
}

func TestOutPool_Basic(t *testing.T) {
	upstream := []string{"127.0.0.1:9999"}
	op := NewOutPool(1, false, nil, nil, 100, 0, 10, upstream)
	if len(op.UpstreamPool) != 1 {
		t.Errorf("Expected 1 upstream pool, got %d", len(op.UpstreamPool))
	}
	_, err := op.GetConnFromConnectionPool("127.0.0.1:8888")
	if err == nil {
		t.Error("Expected error for unknown address")
	}
	_, err = op.GetConnFromConnectionPool("127.0.0.1:9999")
	if err == nil {
		t.Error("Expected error from empty pool (as connections will fail)")
	}
	_, err = op.GetConnFromConnectionPool("127.0.0.1:9999")
	if err == nil {
		t.Error("Expected error from empty pool (as connections will fail)")
	}
}
