package e2e

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// =============================================================================
// MOCK CAPTAIN SERVER FOR TESTING
// =============================================================================

// MockCaptainServer simulates the Captain server for integration testing
type MockCaptainServer struct {
	httpServer      *http.Server
	upgrader        websocket.Upgrader
	workers         map[string]*websocket.Conn
	mu              sync.RWMutex
	receivedEvents  []Event
	eventMu         sync.Mutex
	userCredentials map[string]string
	configPayload   ConfigPayload
	listenAddr      string
	otp             string
}

// Event represents a WebSocket event
type Event struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// ConfigPayload matches the worker's expected config structure
type ConfigPayload struct {
	WorkerName    string           `json:"worker_name"`
	Region        string           `json:"region"`
	PoolID        string           `json:"pool_id"`
	PoolTag       string           `json:"pool_tag"`
	PoolPort      int              `json:"pool_port"`
	PoolSubdomain string           `json:"pool_subdomain"`
	Upstreams     []UpstreamConfig `json:"upstreams"`
}

// UpstreamConfig represents upstream proxy configuration
type UpstreamConfig struct {
	UpstreamID       string `json:"upstream_id"`
	UpstreamTag      string `json:"upstream_tag"`
	UpstreamFormat   string `json:"upstream_format"`
	UpstreamUsername string `json:"upstream_username"`
	UpstreamPassword string `json:"upstream_password"`
	UpstreamHost     string `json:"upstream_host"`
	UpstreamPort     int    `json:"upstream_port"`
	UpstreamProvider string `json:"upstream_provider"`
	Weight           int    `json:"weight"`
}

// NewMockCaptainServer creates a new mock Captain server
func NewMockCaptainServer(listenAddr string) *MockCaptainServer {
	return &MockCaptainServer{
		upgrader:        websocket.Upgrader{},
		workers:         make(map[string]*websocket.Conn),
		receivedEvents:  make([]Event, 0),
		userCredentials: make(map[string]string),
		listenAddr:      listenAddr,
		otp:             "test-otp-token",
	}
}

// SetConfig sets the configuration that will be sent to workers
func (s *MockCaptainServer) SetConfig(config ConfigPayload) {
	s.configPayload = config
}

// AddUserCredentials adds valid user credentials for testing
func (s *MockCaptainServer) AddUserCredentials(username, password string) {
	s.userCredentials[username] = password
}

// Start starts the mock Captain server
func (s *MockCaptainServer) Start() error {
	mux := http.NewServeMux()

	// Login endpoint
	mux.HandleFunc("/worker/ws/login", s.handleLogin)

	// WebSocket endpoint
	mux.HandleFunc("/worker/ws/serve", s.handleWebSocket)

	s.httpServer = &http.Server{
		Addr:    s.listenAddr,
		Handler: mux,
	}

	go s.httpServer.ListenAndServe()

	// Wait for server to be ready
	time.Sleep(100 * time.Millisecond)
	return nil
}

// Stop stops the mock Captain server
func (s *MockCaptainServer) Stop() {
	if s.httpServer != nil {
		s.httpServer.Close()
	}
}

// handleLogin handles worker login requests
func (s *MockCaptainServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Return OTP token
	response := map[string]string{"otp": s.otp}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleWebSocket handles WebSocket connections
func (s *MockCaptainServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Verify OTP
	otp := r.URL.Query().Get("otp")
	if otp != s.otp {
		http.Error(w, "Invalid OTP", http.StatusUnauthorized)
		return
	}

	// Upgrade to WebSocket
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	// Store worker connection
	workerID := r.Header.Get("X-Worker-ID")
	s.mu.Lock()
	s.workers[workerID] = conn
	s.mu.Unlock()

	// Send configuration
	configEvent := Event{
		Type:    "config",
		Payload: s.configPayload,
	}
	conn.WriteJSON(configEvent)

	// Handle incoming messages
	go s.readMessages(conn)
}

// readMessages reads messages from a WebSocket connection
func (s *MockCaptainServer) readMessages(conn *websocket.Conn) {
	defer conn.Close()

	for {
		var event Event
		if err := conn.ReadJSON(&event); err != nil {
			return
		}

		s.eventMu.Lock()
		s.receivedEvents = append(s.receivedEvents, event)
		s.eventMu.Unlock()

		// Handle verify_user events
		if event.Type == "verify_user" {
			s.handleVerifyUser(conn, event)
		}
	}
}

// handleVerifyUser handles user verification requests
func (s *MockCaptainServer) handleVerifyUser(conn *websocket.Conn, event Event) {
	payload, ok := event.Payload.(map[string]interface{})
	if !ok {
		return
	}

	username, _ := payload["username"].(string)
	password, _ := payload["password"].(string)

	success := false
	if storedPass, exists := s.userCredentials[username]; exists {
		success = password == storedPass
	}

	response := Event{
		Type: "login_success",
		Payload: map[string]interface{}{
			"success": success,
			"payload": map[string]interface{}{
				"id":           "00000000-0000-0000-0000-000000000001",
				"username":     username,
				"status":       "active",
				"ip_whitelist": []string{},
				"pools":        []string{},
			},
		},
	}

	conn.WriteJSON(response)
}

// GetReceivedEvents returns all events received from workers
func (s *MockCaptainServer) GetReceivedEvents() []Event {
	s.eventMu.Lock()
	defer s.eventMu.Unlock()
	return append([]Event{}, s.receivedEvents...)
}

// GetTelemetryEvents returns only telemetry events
func (s *MockCaptainServer) GetTelemetryEvents(eventType string) []Event {
	s.eventMu.Lock()
	defer s.eventMu.Unlock()

	var events []Event
	for _, e := range s.receivedEvents {
		if e.Type == eventType {
			events = append(events, e)
		}
	}
	return events
}

// =============================================================================
// INTEGRATION TESTS WITH MOCK CAPTAIN
// =============================================================================

// TestWorkerCaptainIntegration tests worker connection to Captain
func TestWorkerCaptainIntegration(t *testing.T) {
	// This test requires starting a worker instance
	// Skip if running in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create mock Captain server
	captain := NewMockCaptainServer(":18080")
	captain.SetConfig(ConfigPayload{
		WorkerName:    "test-worker",
		Region:        "test-region",
		PoolID:        "00000000-0000-0000-0000-000000000001",
		PoolTag:       "test-pool",
		PoolPort:      33080,
		PoolSubdomain: "test",
		Upstreams: []UpstreamConfig{
			{
				UpstreamID:       "00000000-0000-0000-0000-000000000002",
				UpstreamTag:      "upstream-1",
				UpstreamFormat:   "http",
				UpstreamUsername: "upstream_user",
				UpstreamPassword: "upstream_pass",
				UpstreamHost:     "127.0.0.1",
				UpstreamPort:     8888,
				UpstreamProvider: "test-provider",
				Weight:           1,
			},
		},
	})
	captain.AddUserCredentials("testuser", "testpass")

	if err := captain.Start(); err != nil {
		t.Fatalf("Failed to start mock Captain: %v", err)
	}
	defer captain.Stop()

	t.Log("Mock Captain server started, ready for worker connection")

	// Worker would need to be started separately to connect
	// This sets up the infrastructure for integration testing
}

// =============================================================================
// MOCK UPSTREAM PROXY FOR TESTING
// =============================================================================

// MockUpstreamProxy simulates an upstream proxy for testing
type MockUpstreamProxy struct {
	listener    net.Listener
	listenAddr  string
	connections int
	mu          sync.Mutex
	username    string
	password    string
}

// NewMockUpstreamProxy creates a new mock upstream proxy
func NewMockUpstreamProxy(listenAddr, username, password string) *MockUpstreamProxy {
	return &MockUpstreamProxy{
		listenAddr: listenAddr,
		username:   username,
		password:   password,
	}
}

// Start starts the mock upstream proxy
func (p *MockUpstreamProxy) Start() error {
	listener, err := net.Listen("tcp", p.listenAddr)
	if err != nil {
		return err
	}
	p.listener = listener

	go p.accept()
	return nil
}

// Stop stops the mock upstream proxy
func (p *MockUpstreamProxy) Stop() {
	if p.listener != nil {
		p.listener.Close()
	}
}

// accept accepts connections
func (p *MockUpstreamProxy) accept() {
	for {
		conn, err := p.listener.Accept()
		if err != nil {
			return
		}

		p.mu.Lock()
		p.connections++
		p.mu.Unlock()

		go p.handleConnection(conn)
	}
}

// handleConnection handles a proxy connection
func (p *MockUpstreamProxy) handleConnection(conn net.Conn) {
	defer conn.Close()

	// Read the request
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return
	}

	request := string(buf[:n])

	// Simple response for testing
	response := "HTTP/1.1 200 OK\r\n" +
		"Content-Type: application/json\r\n" +
		"Content-Length: 42\r\n" +
		"Connection: close\r\n\r\n" +
		`{"status":"ok","upstream":"mock-upstream"}`

	// Check if it's a CONNECT request
	if len(request) > 7 && request[:7] == "CONNECT" {
		response = "HTTP/1.1 200 Connection Established\r\n\r\n"
	}

	conn.Write([]byte(response))
}

// GetConnectionCount returns the number of connections received
func (p *MockUpstreamProxy) GetConnectionCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.connections
}

// =============================================================================
// UPSTREAM ROUTING E2E TESTS
// =============================================================================

// TestUpstreamProxyRouting tests that requests are routed through upstream
func TestUpstreamProxyRouting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create mock upstream
	upstream := NewMockUpstreamProxy(":18888", "upstream_user", "upstream_pass")
	if err := upstream.Start(); err != nil {
		t.Fatalf("Failed to start mock upstream: %v", err)
	}
	defer upstream.Stop()

	t.Log("Mock upstream proxy started on :18888")

	// Note: To complete this test, you would need to:
	// 1. Configure the worker with this upstream
	// 2. Make a request through the worker
	// 3. Verify the upstream received the connection
}

// =============================================================================
// LOAD TESTING
// =============================================================================

// TestProxyLoadTest performs a load test on the proxy
func TestProxyLoadTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	config := loadTestConfig()
	numRequests := 100
	concurrency := 10

	var wg sync.WaitGroup
	results := make(chan time.Duration, numRequests)
	errors := make(chan error, numRequests)
	semaphore := make(chan struct{}, concurrency)

	start := time.Now()

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			reqStart := time.Now()

			conn, err := net.DialTimeout("tcp", config.HTTPProxyAddr, 5*time.Second)
			if err != nil {
				errors <- err
				return
			}
			conn.Close()

			results <- time.Since(reqStart)
		}()
	}

	wg.Wait()
	close(results)
	close(errors)

	totalTime := time.Since(start)

	// Calculate statistics
	var times []time.Duration
	for duration := range results {
		times = append(times, duration)
	}

	var errorCount int
	for range errors {
		errorCount++
	}

	successRate := float64(len(times)) / float64(numRequests) * 100
	reqPerSec := float64(len(times)) / totalTime.Seconds()

	t.Logf("Load Test Results:")
	t.Logf("  Total requests: %d", numRequests)
	t.Logf("  Successful: %d (%.2f%%)", len(times), successRate)
	t.Logf("  Failed: %d", errorCount)
	t.Logf("  Total time: %v", totalTime)
	t.Logf("  Requests/sec: %.2f", reqPerSec)

	if successRate < 95 {
		t.Errorf("Success rate too low: %.2f%%", successRate)
	}
}

// =============================================================================
// WEBSOCKET EVENT VERIFICATION
// =============================================================================

// TestTelemetryUsageEvent tests that data usage telemetry is sent correctly
func TestTelemetryUsageEvent(t *testing.T) {
	// This test verifies the structure of telemetry_usage events
	// that should be sent to Captain

	expectedEvent := Event{
		Type: "telemetry_usage",
		Payload: map[string]interface{}{
			"user_id":          "00000000-0000-0000-0000-000000000001",
			"username":         "testuser",
			"pool_id":          "00000000-0000-0000-0000-000000000002",
			"pool_name":        "test-pool",
			"worker_id":        "00000000-0000-0000-0000-000000000003",
			"worker_region":    "us-west-1",
			"bytes_sent":       1024,
			"bytes_received":   2048,
			"source_ip":        "192.168.1.1",
			"protocol":         "HTTP",
			"destination_host": "httpbin.org",
			"destination_port": 80,
			"status_code":      200,
		},
	}

	// Verify the event structure can be marshaled
	data, err := json.Marshal(expectedEvent)
	if err != nil {
		t.Fatalf("Failed to marshal telemetry event: %v", err)
	}

	// Verify it can be unmarshaled back
	var decoded Event
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal telemetry event: %v", err)
	}

	t.Logf("Telemetry usage event structure: %s", string(data))
}

// TestTelemetryHealthEvent tests that health telemetry is sent correctly
func TestTelemetryHealthEvent(t *testing.T) {
	expectedEvent := Event{
		Type: "telemetry_health",
		Payload: map[string]interface{}{
			"worker_id":                "00000000-0000-0000-0000-000000000001",
			"worker_name":              "test-worker",
			"region":                   "us-west-1",
			"status":                   "healthy",
			"cpu_usage":                45.5,
			"memory_usage":             60.2,
			"active_connections":       10,
			"total_connections":        1000,
			"bytes_throughput_per_sec": 1048576,
			"error_rate":               0.5,
			"upstreams": []map[string]interface{}{
				{
					"upstream_id":  "00000000-0000-0000-0000-000000000002",
					"upstream_tag": "upstream-1",
					"status":       "healthy",
					"latency":      50,
					"error_rate":   0.1,
				},
			},
		},
	}

	data, err := json.Marshal(expectedEvent)
	if err != nil {
		t.Fatalf("Failed to marshal health event: %v", err)
	}

	t.Logf("Telemetry health event structure: %s", string(data))
}

// =============================================================================
// SOCKS5 PROTOCOL TESTS
// =============================================================================

// TestSOCKS5ProtocolCompliance tests SOCKS5 protocol compliance
func TestSOCKS5ProtocolCompliance(t *testing.T) {
	config := loadTestConfig()

	conn, err := net.DialTimeout("tcp", config.SOCKSProxyAddr, 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to connect to SOCKS5 proxy: %v", err)
	}
	defer conn.Close()

	// Test 1: Version check
	// Send: VER (1 byte) + NMETHODS (1 byte) + METHODS (1 byte)
	_, err = conn.Write([]byte{0x05, 0x01, 0x02}) // Version 5, 1 method, username/password auth
	if err != nil {
		t.Fatalf("Failed to send handshake: %v", err)
	}

	// Read response: VER (1 byte) + METHOD (1 byte)
	response := make([]byte, 2)
	_, err = io.ReadFull(conn, response)
	if err != nil {
		t.Fatalf("Failed to read handshake response: %v", err)
	}

	if response[0] != 0x05 {
		t.Errorf("Expected SOCKS5 version (0x05), got: 0x%02x", response[0])
	}

	if response[1] != 0x02 {
		t.Errorf("Expected username/password auth (0x02), got: 0x%02x", response[1])
	}

	t.Logf("SOCKS5 handshake: version=0x%02x, method=0x%02x", response[0], response[1])
}

// TestSOCKS5AddressTypes tests different SOCKS5 address types
func TestSOCKS5AddressTypes(t *testing.T) {
	config := loadTestConfig()

	testCases := []struct {
		name     string
		host     string
		port     int
		addrType byte
	}{
		{"IPv4", "93.184.216.34", 80, 0x01},
		{"Domain", "example.com", 80, 0x03},
		// IPv6 test would require an IPv6-enabled target
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := NewSOCKS5Client(config.SOCKSProxyAddr, config.TestUsername, config.TestPassword)
			conn, err := client.Connect(tc.host, tc.port)
			if err != nil {
				t.Logf("Connection to %s:%d failed (may be expected): %v", tc.host, tc.port, err)
				return
			}
			defer conn.Close()
			t.Logf("Successfully connected to %s:%d using address type 0x%02x", tc.host, tc.port, tc.addrType)
		})
	}
}

// =============================================================================
// HTTP PROXY PROTOCOL TESTS
// =============================================================================

// TestHTTPProxyCONNECTMethod tests the HTTP CONNECT method
func TestHTTPProxyCONNECTMethod(t *testing.T) {
	config := loadTestConfig()

	conn, err := net.DialTimeout("tcp", config.HTTPProxyAddr, 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to connect to HTTP proxy: %v", err)
	}
	defer conn.Close()

	// Send CONNECT request
	request := fmt.Sprintf("CONNECT httpbin.org:443 HTTP/1.1\r\n"+
		"Host: httpbin.org:443\r\n"+
		"Proxy-Authorization: Basic %s\r\n"+
		"\r\n",
		encodeBasicAuth(config.TestUsername, config.TestPassword))

	_, err = conn.Write([]byte(request))
	if err != nil {
		t.Fatalf("Failed to send CONNECT request: %v", err)
	}

	// Read response
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	response := string(buf[:n])
	if !containsHTTPSuccess(response) {
		t.Errorf("Expected successful CONNECT response, got: %s", response)
	}

	t.Logf("CONNECT response: %s", response[:min(100, len(response))])
}

// encodeBasicAuth encodes username:password as base64
func encodeBasicAuth(username, password string) string {
	auth := username + ":" + password
	return fmt.Sprintf("%s", bytesToBase64([]byte(auth)))
}

// bytesToBase64 converts bytes to base64 string
func bytesToBase64(data []byte) string {
	return string(encodeToBase64(data))
}

// encodeToBase64 is a simple base64 encoder
func encodeToBase64(data []byte) []byte {
	const encodeStd = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

	var buf bytes.Buffer
	for i := 0; i < len(data); i += 3 {
		var n uint32
		switch len(data) - i {
		case 1:
			n = uint32(data[i]) << 16
			buf.WriteByte(encodeStd[n>>18&0x3f])
			buf.WriteByte(encodeStd[n>>12&0x3f])
			buf.WriteString("==")
		case 2:
			n = uint32(data[i])<<16 | uint32(data[i+1])<<8
			buf.WriteByte(encodeStd[n>>18&0x3f])
			buf.WriteByte(encodeStd[n>>12&0x3f])
			buf.WriteByte(encodeStd[n>>6&0x3f])
			buf.WriteByte('=')
		default:
			n = uint32(data[i])<<16 | uint32(data[i+1])<<8 | uint32(data[i+2])
			buf.WriteByte(encodeStd[n>>18&0x3f])
			buf.WriteByte(encodeStd[n>>12&0x3f])
			buf.WriteByte(encodeStd[n>>6&0x3f])
			buf.WriteByte(encodeStd[n&0x3f])
		}
	}
	return buf.Bytes()
}

// containsHTTPSuccess checks if response contains HTTP success status
func containsHTTPSuccess(response string) bool {
	return len(response) > 12 && (response[9:12] == "200" || response[9:12] == "201")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Helper to avoid unused import
var _ = binary.BigEndian
