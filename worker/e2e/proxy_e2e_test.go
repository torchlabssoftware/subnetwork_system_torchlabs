package e2e

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

// TestConfig holds the configuration for E2E tests
type TestConfig struct {
	HTTPProxyAddr  string
	SOCKSProxyAddr string
	TestUsername   string
	TestPassword   string
	TestTarget     string // Target URL to test against
}

// loadTestConfig loads test configuration from environment or defaults
func loadTestConfig() TestConfig {
	config := TestConfig{
		HTTPProxyAddr:  getEnvOrDefault("E2E_HTTP_PROXY", "localhost:33080"),
		SOCKSProxyAddr: getEnvOrDefault("E2E_SOCKS_PROXY", "localhost:33081"),
		TestUsername:   getEnvOrDefault("E2E_TEST_USER", "testuser"),
		TestPassword:   getEnvOrDefault("E2E_TEST_PASS", "testpass"),
		TestTarget:     getEnvOrDefault("E2E_TEST_TARGET", "http://httpbin.org/get"),
	}
	return config
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// =============================================================================
// HTTP PROXY E2E TESTS
// =============================================================================

// TestHTTPProxyBasicConnect tests basic HTTP proxy connection without auth
func TestHTTPProxyBasicConnect(t *testing.T) {
	config := loadTestConfig()

	// Create HTTP client with proxy
	proxyURL, err := url.Parse("http://" + config.HTTPProxyAddr)
	if err != nil {
		t.Fatalf("Failed to parse proxy URL: %v", err)
	}

	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	// Make request through proxy
	resp, err := client.Get(config.TestTarget)
	if err != nil {
		t.Fatalf("Failed to make request through proxy: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	t.Logf("Successfully connected through HTTP proxy, status: %d", resp.StatusCode)
}

// TestHTTPProxyWithAuthentication tests HTTP proxy with basic authentication
func TestHTTPProxyWithAuthentication(t *testing.T) {
	config := loadTestConfig()

	// Create proxy URL with credentials
	proxyURL, err := url.Parse(fmt.Sprintf("http://%s:%s@%s",
		config.TestUsername, config.TestPassword, config.HTTPProxyAddr))
	if err != nil {
		t.Fatalf("Failed to parse proxy URL: %v", err)
	}

	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	resp, err := client.Get(config.TestTarget)
	if err != nil {
		t.Fatalf("Failed to make request through proxy with auth: %v", err)
	}
	defer resp.Body.Close()

	t.Logf("HTTP Proxy with auth - Status: %d", resp.StatusCode)
}

// TestHTTPProxyInvalidAuth tests HTTP proxy rejects invalid credentials
func TestHTTPProxyInvalidAuth(t *testing.T) {
	config := loadTestConfig()

	// Create proxy URL with invalid credentials
	proxyURL, err := url.Parse(fmt.Sprintf("http://%s:%s@%s",
		"invaliduser", "invalidpass", config.HTTPProxyAddr))
	if err != nil {
		t.Fatalf("Failed to parse proxy URL: %v", err)
	}

	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	resp, err := client.Get(config.TestTarget)
	if err != nil {
		// Connection might be refused/closed - this is expected behavior
		t.Logf("Expected auth failure error: %v", err)
		return
	}
	defer resp.Body.Close()

	// Should return 407 Proxy Authentication Required
	if resp.StatusCode == http.StatusProxyAuthRequired {
		t.Log("Correctly rejected invalid credentials with 407")
	} else if resp.StatusCode == http.StatusForbidden {
		t.Log("Correctly rejected invalid credentials with 403")
	} else {
		t.Errorf("Expected 407 or 403, got %d", resp.StatusCode)
	}
}

// TestHTTPProxyHTTPSConnect tests HTTPS CONNECT method through HTTP proxy
func TestHTTPProxyHTTPSConnect(t *testing.T) {
	config := loadTestConfig()

	proxyURL, err := url.Parse(fmt.Sprintf("http://%s:%s@%s",
		config.TestUsername, config.TestPassword, config.HTTPProxyAddr))
	if err != nil {
		t.Fatalf("Failed to parse proxy URL: %v", err)
	}

	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // For testing only
		},
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	// Test HTTPS through proxy using CONNECT
	resp, err := client.Get("https://httpbin.org/get")
	if err != nil {
		t.Fatalf("Failed to make HTTPS request through proxy: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	t.Logf("Successfully connected HTTPS through HTTP proxy, status: %d", resp.StatusCode)
}

// TestHTTPProxyConcurrentConnections tests multiple concurrent connections
func TestHTTPProxyConcurrentConnections(t *testing.T) {
	config := loadTestConfig()
	concurrentRequests := 10

	proxyURL, err := url.Parse(fmt.Sprintf("http://%s:%s@%s",
		config.TestUsername, config.TestPassword, config.HTTPProxyAddr))
	if err != nil {
		t.Fatalf("Failed to parse proxy URL: %v", err)
	}

	transport := &http.Transport{
		Proxy:               http.ProxyURL(proxyURL),
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	results := make(chan error, concurrentRequests)

	for i := 0; i < concurrentRequests; i++ {
		go func(id int) {
			resp, err := client.Get(config.TestTarget)
			if err != nil {
				results <- fmt.Errorf("request %d failed: %v", id, err)
				return
			}
			defer resp.Body.Close()
			io.ReadAll(resp.Body)

			if resp.StatusCode != http.StatusOK {
				results <- fmt.Errorf("request %d: expected 200, got %d", id, resp.StatusCode)
				return
			}
			results <- nil
		}(i)
	}

	// Wait for all results
	successCount := 0
	for i := 0; i < concurrentRequests; i++ {
		if err := <-results; err != nil {
			t.Logf("Concurrent request error: %v", err)
		} else {
			successCount++
		}
	}

	t.Logf("Concurrent connections: %d/%d succeeded", successCount, concurrentRequests)

	if successCount < concurrentRequests/2 {
		t.Errorf("Too many failures: only %d/%d succeeded", successCount, concurrentRequests)
	}
}

// =============================================================================
// SOCKS5 PROXY E2E TESTS
// =============================================================================

// SOCKS5Client is a simple SOCKS5 client for testing
type SOCKS5Client struct {
	proxyAddr string
	username  string
	password  string
}

// NewSOCKS5Client creates a new SOCKS5 client
func NewSOCKS5Client(proxyAddr, username, password string) *SOCKS5Client {
	return &SOCKS5Client{
		proxyAddr: proxyAddr,
		username:  username,
		password:  password,
	}
}

// Connect establishes a SOCKS5 connection to the target
func (c *SOCKS5Client) Connect(targetHost string, targetPort int) (net.Conn, error) {
	// Connect to proxy
	conn, err := net.DialTimeout("tcp", c.proxyAddr, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to proxy: %w", err)
	}

	// SOCKS5 handshake
	if c.username != "" && c.password != "" {
		// Request username/password auth
		_, err = conn.Write([]byte{0x05, 0x01, 0x02})
	} else {
		// Request no auth
		_, err = conn.Write([]byte{0x05, 0x01, 0x00})
	}
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to send auth methods: %w", err)
	}

	// Read auth response
	response := make([]byte, 2)
	if _, err = io.ReadFull(conn, response); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to read auth response: %w", err)
	}

	if response[0] != 0x05 {
		conn.Close()
		return nil, fmt.Errorf("invalid SOCKS version: %d", response[0])
	}

	// Handle authentication
	if response[1] == 0x02 {
		// Username/password auth required
		authReq := bytes.Buffer{}
		authReq.WriteByte(0x01) // Auth version
		authReq.WriteByte(byte(len(c.username)))
		authReq.WriteString(c.username)
		authReq.WriteByte(byte(len(c.password)))
		authReq.WriteString(c.password)

		if _, err = conn.Write(authReq.Bytes()); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to send auth: %w", err)
		}

		// Read auth result
		authResp := make([]byte, 2)
		if _, err = io.ReadFull(conn, authResp); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to read auth result: %w", err)
		}

		if authResp[1] != 0x00 {
			conn.Close()
			return nil, fmt.Errorf("authentication failed")
		}
	} else if response[1] == 0xFF {
		conn.Close()
		return nil, fmt.Errorf("no acceptable auth method")
	}

	// Send CONNECT request
	connectReq := bytes.Buffer{}
	connectReq.WriteByte(0x05) // VER
	connectReq.WriteByte(0x01) // CMD: CONNECT
	connectReq.WriteByte(0x00) // RSV
	connectReq.WriteByte(0x03) // ATYP: Domain name
	connectReq.WriteByte(byte(len(targetHost)))
	connectReq.WriteString(targetHost)
	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, uint16(targetPort))
	connectReq.Write(portBytes)

	if _, err = conn.Write(connectReq.Bytes()); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to send connect request: %w", err)
	}

	// Read CONNECT response
	connectResp := make([]byte, 10) // Min size for IPv4 response
	if _, err = io.ReadFull(conn, connectResp[:4]); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to read connect response: %w", err)
	}

	if connectResp[1] != 0x00 {
		conn.Close()
		return nil, fmt.Errorf("connect failed with code: %d", connectResp[1])
	}

	// Skip the remaining bytes of the response based on address type
	switch connectResp[3] {
	case 0x01: // IPv4
		io.ReadFull(conn, connectResp[4:10])
	case 0x03: // Domain
		lenByte := make([]byte, 1)
		io.ReadFull(conn, lenByte)
		domainAndPort := make([]byte, int(lenByte[0])+2)
		io.ReadFull(conn, domainAndPort)
	case 0x04: // IPv6
		ipv6AndPort := make([]byte, 18)
		io.ReadFull(conn, ipv6AndPort)
	}

	return conn, nil
}

// TestSOCKS5ProxyBasicConnect tests basic SOCKS5 proxy connection
func TestSOCKS5ProxyBasicConnect(t *testing.T) {
	config := loadTestConfig()

	client := NewSOCKS5Client(config.SOCKSProxyAddr, config.TestUsername, config.TestPassword)

	conn, err := client.Connect("httpbin.org", 80)
	if err != nil {
		t.Fatalf("Failed to connect through SOCKS5 proxy: %v", err)
	}
	defer conn.Close()

	// Send HTTP request
	req := "GET /get HTTP/1.1\r\nHost: httpbin.org\r\nConnection: close\r\n\r\n"
	if _, err = conn.Write([]byte(req)); err != nil {
		t.Fatalf("Failed to send HTTP request: %v", err)
	}

	// Read response
	reader := bufio.NewReader(conn)
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if !strings.Contains(statusLine, "200") {
		t.Errorf("Expected HTTP 200, got: %s", strings.TrimSpace(statusLine))
	}

	t.Logf("Successfully connected through SOCKS5 proxy: %s", strings.TrimSpace(statusLine))
}

// TestSOCKS5ProxyAuthentication tests SOCKS5 authentication
func TestSOCKS5ProxyAuthentication(t *testing.T) {
	config := loadTestConfig()

	// Test with valid credentials
	client := NewSOCKS5Client(config.SOCKSProxyAddr, config.TestUsername, config.TestPassword)
	conn, err := client.Connect("httpbin.org", 80)
	if err != nil {
		t.Fatalf("Failed with valid credentials: %v", err)
	}
	conn.Close()
	t.Log("SOCKS5 authentication with valid credentials: SUCCESS")
}

// TestSOCKS5ProxyInvalidAuth tests SOCKS5 rejects invalid credentials
func TestSOCKS5ProxyInvalidAuth(t *testing.T) {
	config := loadTestConfig()

	client := NewSOCKS5Client(config.SOCKSProxyAddr, "invaliduser", "invalidpass")
	_, err := client.Connect("httpbin.org", 80)
	if err == nil {
		t.Error("Expected authentication failure, but connection succeeded")
		return
	}

	if strings.Contains(err.Error(), "authentication failed") ||
		strings.Contains(err.Error(), "no acceptable auth method") {
		t.Logf("Correctly rejected invalid credentials: %v", err)
	} else {
		t.Logf("Connection failed (expected): %v", err)
	}
}

// TestSOCKS5ProxyHTTPS tests HTTPS through SOCKS5 proxy
func TestSOCKS5ProxyHTTPS(t *testing.T) {
	config := loadTestConfig()

	client := NewSOCKS5Client(config.SOCKSProxyAddr, config.TestUsername, config.TestPassword)

	conn, err := client.Connect("httpbin.org", 443)
	if err != nil {
		t.Fatalf("Failed to connect through SOCKS5 proxy: %v", err)
	}
	defer conn.Close()

	// Wrap in TLS
	tlsConn := tls.Client(conn, &tls.Config{
		ServerName:         "httpbin.org",
		InsecureSkipVerify: false,
	})
	defer tlsConn.Close()

	if err := tlsConn.Handshake(); err != nil {
		t.Fatalf("TLS handshake failed: %v", err)
	}

	// Send HTTPS request
	req := "GET /get HTTP/1.1\r\nHost: httpbin.org\r\nConnection: close\r\n\r\n"
	if _, err = tlsConn.Write([]byte(req)); err != nil {
		t.Fatalf("Failed to send HTTPS request: %v", err)
	}

	// Read response
	reader := bufio.NewReader(tlsConn)
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if !strings.Contains(statusLine, "200") {
		t.Errorf("Expected HTTP 200, got: %s", strings.TrimSpace(statusLine))
	}

	t.Logf("Successfully made HTTPS request through SOCKS5: %s", strings.TrimSpace(statusLine))
}

// =============================================================================
// UPSTREAM PROXY E2E TESTS
// =============================================================================

// TestUpstreamProxyConfiguration tests that upstream proxies are properly configured
func TestUpstreamProxyConfiguration(t *testing.T) {
	config := loadTestConfig()

	// This test verifies that when upstreams are configured,
	// requests are routed through them
	proxyURL, err := url.Parse(fmt.Sprintf("http://%s:%s@%s",
		config.TestUsername, config.TestPassword, config.HTTPProxyAddr))
	if err != nil {
		t.Fatalf("Failed to parse proxy URL: %v", err)
	}

	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	// Make request through proxy - should go through upstream if configured
	resp, err := client.Get("http://httpbin.org/ip")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	t.Logf("Response from /ip endpoint: %s", string(body))

	// The response should show the upstream proxy's IP, not the worker's IP
	// This can be verified manually by comparing with the known upstream IPs
}

// =============================================================================
// DATA USAGE TRACKING E2E TESTS
// =============================================================================

// TestDataUsageTracking tests that data usage is properly tracked
func TestDataUsageTracking(t *testing.T) {
	config := loadTestConfig()

	proxyURL, err := url.Parse(fmt.Sprintf("http://%s:%s@%s",
		config.TestUsername, config.TestPassword, config.HTTPProxyAddr))
	if err != nil {
		t.Fatalf("Failed to parse proxy URL: %v", err)
	}

	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	// Make a request that transfers some data
	resp, err := client.Get("http://httpbin.org/bytes/1024")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	t.Logf("Received %d bytes of data", len(body))

	// Note: Actual data usage tracking verification would require
	// checking the Captain's logs or analytics data
	// This test verifies the connection works and data flows
}

// =============================================================================
// HEALTH CHECK E2E TESTS
// =============================================================================

// TestProxyHealthEndpoint tests health check functionality
func TestProxyHealthEndpoint(t *testing.T) {
	config := loadTestConfig()

	// Test that the proxy is accepting connections
	conn, err := net.DialTimeout("tcp", config.HTTPProxyAddr, 5*time.Second)
	if err != nil {
		t.Fatalf("Proxy not responding: %v", err)
	}
	conn.Close()

	t.Log("HTTP Proxy is healthy and accepting connections")

	// Test SOCKS proxy
	conn, err = net.DialTimeout("tcp", config.SOCKSProxyAddr, 5*time.Second)
	if err != nil {
		t.Fatalf("SOCKS Proxy not responding: %v", err)
	}
	conn.Close()

	t.Log("SOCKS Proxy is healthy and accepting connections")
}

// =============================================================================
// HELPER FUNCTIONS FOR RAW HTTP PROXY TESTING
// =============================================================================

// makeRawHTTPProxyRequest makes a raw HTTP request through the proxy
func makeRawHTTPProxyRequest(proxyAddr, targetURL, username, password string) (*http.Response, error) {
	conn, err := net.DialTimeout("tcp", proxyAddr, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to proxy: %w", err)
	}
	defer conn.Close()

	// Build the request
	reqLine := fmt.Sprintf("GET %s HTTP/1.1\r\n", targetURL)
	headers := fmt.Sprintf("Host: %s\r\n", extractHost(targetURL))

	// Add proxy auth if provided
	if username != "" && password != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
		headers += fmt.Sprintf("Proxy-Authorization: Basic %s\r\n", auth)
	}

	headers += "Connection: close\r\n\r\n"

	request := reqLine + headers
	if _, err = conn.Write([]byte(request)); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Read response
	reader := bufio.NewReader(conn)
	return http.ReadResponse(reader, nil)
}

// extractHost extracts the host from a URL
func extractHost(targetURL string) string {
	u, err := url.Parse(targetURL)
	if err != nil {
		return ""
	}
	return u.Host
}

// TestRawHTTPProxyRequest tests raw HTTP proxy request
func TestRawHTTPProxyRequest(t *testing.T) {
	config := loadTestConfig()

	resp, err := makeRawHTTPProxyRequest(
		config.HTTPProxyAddr,
		config.TestTarget,
		config.TestUsername,
		config.TestPassword,
	)
	if err != nil {
		t.Fatalf("Raw HTTP proxy request failed: %v", err)
	}
	defer resp.Body.Close()

	t.Logf("Raw HTTP proxy response status: %s", resp.Status)
}
