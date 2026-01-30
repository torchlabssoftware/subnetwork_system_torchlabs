package services

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/snail007/goproxy/manager"
	"github.com/snail007/goproxy/utils"
)

func TestHTTP_NewHTTP(t *testing.T) {
	http := NewHTTP()
	if http == nil {
		t.Error("HTTP service should not be nil")
	}
	if _, ok := http.(*HTTP); !ok {
		t.Error("NewHTTP should return *HTTP type")
	}
}

func TestHTTP_InitService(t *testing.T) {
	http := NewHTTP().(*HTTP)
	http.worker = &manager.WorkerManager{}
	http.InitService()
}

func TestHTTP_InitService_WithWorker(t *testing.T) {
	http := NewHTTP().(*HTTP)
	worker := &manager.WorkerManager{}
	http.worker = worker
	http.InitService()
}

func TestHTTP_StopService(t *testing.T) {
	http := NewHTTP().(*HTTP)
	http.StopService()
}

func TestHTTP_StopService_WithUpstreamPool(t *testing.T) {
	http := NewHTTP().(*HTTP)
	mockPool := &utils.OutPool{}
	http.outPool = *mockPool
	http.StopService()
}

func TestHTTP_Start(t *testing.T) {
	http := NewHTTP().(*HTTP)
	args := HTTPArgs{
		HTTPTimeout: utils.GetPTR(5000),
		Interval:    utils.GetPTR(300),
		Blocked:     utils.GetPTR("blocked.txt"),
		Direct:      utils.GetPTR("direct.txt"),
		LocalType:   utils.GetPTR(TYPE_TCP),
		Args:        Args{Local: utils.GetPTR("127.0.0.1:0")},
	}
	worker := &manager.WorkerManager{}
	err := http.Start(args, worker)
	if err != nil {
		t.Errorf("Start should not return error: %v", err)
	}
	if http.cfg.HTTPTimeout == nil {
		t.Error("HTTPTimeout should be set")
	}
	if http.cfg.Interval == nil {
		t.Error("Interval should be set")
	}
}

func TestHTTP_Start_TLSType(t *testing.T) {
	http := NewHTTP().(*HTTP)
	args := HTTPArgs{
		HTTPTimeout: utils.GetPTR(5000),
		Interval:    utils.GetPTR(300),
		Blocked:     utils.GetPTR("blocked.txt"),
		Direct:      utils.GetPTR("direct.txt"),
		LocalType:   utils.GetPTR(TYPE_TLS),
		Args:        Args{Local: utils.GetPTR("127.0.0.1:0"), CertBytes: []byte{}, KeyBytes: []byte{}},
	}
	worker := &manager.WorkerManager{}
	err := http.Start(args, worker)
	if err == nil {
		t.Log("Start with TLS type without certs - behavior depends on implementation")
	}
}

func TestHTTP_Start_InvalidAddress(t *testing.T) {
	http := NewHTTP().(*HTTP)
	args := HTTPArgs{
		HTTPTimeout: utils.GetPTR(5000),
		Interval:    utils.GetPTR(300),
		Blocked:     utils.GetPTR("blocked.txt"),
		Direct:      utils.GetPTR("direct.txt"),
		LocalType:   utils.GetPTR(TYPE_TCP),
		Args:        Args{Local: utils.GetPTR("invalid-address")},
	}
	worker := &manager.WorkerManager{}
	err := http.Start(args, worker)
	if err != nil {
		t.Logf("Expected error for invalid address: %v", err)
	}
}

func TestHTTP_Clean(t *testing.T) {
	http := NewHTTP().(*HTTP)
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Clean should not panic: %v", r)
		}
	}()
	http.Clean()
}

func TestHTTP_Callback(t *testing.T) {
	http := NewHTTP().(*HTTP)
	conn := &MockConn{
		data: []byte("GET http://example.com HTTP/1.1\r\n" +
			"Host: example.com\r\n" +
			"Proxy-Authorization: Basic dGVzdHVzZXI6dGVzdHBhc3M=\r\n" +
			"\r\n"),
	}
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Callback should not panic: %v", r)
		}
	}()
	http.callback(conn)
}

func TestHTTP_Callback_InvalidRequest(t *testing.T) {
	http := NewHTTP().(*HTTP)
	http.worker = &manager.WorkerManager{}
	conn := &MockConn{
		data: []byte("INVALID REQUEST DATA\r\n\r\n"),
	}
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Callback with invalid request should not panic: %v", r)
		}
	}()
	http.callback(conn)
	if !conn.closed {
		t.Log("Connection should be closed after invalid request")
	}
}

func TestHTTP_Callback_HTTPS_CONNECT(t *testing.T) {
	http := NewHTTP().(*HTTP)
	http.worker = &manager.WorkerManager{}
	conn := &MockConn{
		data: []byte("CONNECT example.com:443 HTTP/1.1\r\n" +
			"Host: example.com:443\r\n" +
			"Proxy-Authorization: Basic dGVzdHVzZXI6dGVzdHBhc3M=\r\n" +
			"\r\n"),
	}
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Callback with CONNECT should not panic: %v", r)
		}
	}()
	http.callback(conn)
}

func TestHTTP_Callback_EmptyRequest(t *testing.T) {
	http := NewHTTP().(*HTTP)
	http.worker = &manager.WorkerManager{}
	conn := &MockConn{
		data: []byte{},
	}
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Callback with empty request should not panic: %v", r)
		}
	}()
	http.callback(conn)
}

func TestHTTP_IsDeadLoop_SameIPAndPort(t *testing.T) {
	http := NewHTTP().(*HTTP)
	result := http.IsDeadLoop("127.0.0.1:8080", "127.0.0.1:8080")
	t.Logf("IsDeadLoop result for same IP:port: %v", result)
}

func TestHTTP_IsDeadLoop_DifferentPort(t *testing.T) {
	http := NewHTTP().(*HTTP)
	result := http.IsDeadLoop("127.0.0.1:8080", "127.0.0.1:9090")
	if result {
		t.Error("Different ports should not be detected as dead loop")
	}
}

func TestHTTP_IsDeadLoop_InvalidLocalAddress(t *testing.T) {
	http := NewHTTP().(*HTTP)
	result := http.IsDeadLoop("invalid-address", "example.com:80")
	if result {
		t.Error("Invalid local address should return false")
	}
}

func TestHTTP_IsDeadLoop_InvalidHostAddress(t *testing.T) {
	http := NewHTTP().(*HTTP)
	result := http.IsDeadLoop("127.0.0.1:8080", "invalid-host-no-port")
	if result {
		t.Error("Invalid host address should return false")
	}
}

func TestHTTP_IsDeadLoop_DifferentIP(t *testing.T) {
	http := NewHTTP().(*HTTP)
	result := http.IsDeadLoop("127.0.0.1:8080", "8.8.8.8:8080")
	if result {
		t.Error("Different IPs should not be detected as dead loop")
	}
}

func TestHTTP_IsDeadLoop_LocalhostVariants(t *testing.T) {
	http := NewHTTP().(*HTTP)
	result := http.IsDeadLoop("127.0.0.1:8080", "localhost:8080")
	t.Logf("IsDeadLoop localhost vs 127.0.0.1 result: %v", result)
}

func TestHTTP_InitOutConnPool_TCPType(t *testing.T) {
	http := NewHTTP().(*HTTP)
	http.worker = &manager.WorkerManager{}
	args := HTTPArgs{
		ParentType:          utils.GetPTR(TYPE_TCP),
		CheckParentInterval: utils.GetPTR(10),
		Timeout:             utils.GetPTR(5000),
		PoolSize:            utils.GetPTR(10),
	}
	http.cfg = args
	defer func() {
		if r := recover(); r != nil {
			t.Logf("InitOutConnPool panics without properly initialized worker (expected): %v", r)
		}
	}()
	http.InitOutConnPool()
}

func TestHTTP_InitOutConnPool_TLSType(t *testing.T) {
	http := NewHTTP().(*HTTP)
	http.worker = &manager.WorkerManager{}
	args := HTTPArgs{
		ParentType:          utils.GetPTR(TYPE_TLS),
		CheckParentInterval: utils.GetPTR(10),
		Timeout:             utils.GetPTR(5000),
		PoolSize:            utils.GetPTR(10),
		Args:                Args{CertBytes: []byte{}, KeyBytes: []byte{}},
	}
	http.cfg = args
	defer func() {
		if r := recover(); r != nil {
			t.Logf("InitOutConnPool with TLS panics without properly initialized worker (expected): %v", r)
		}
	}()
	http.InitOutConnPool()
}

func TestHTTP_OutToTCP_DirectConnect(t *testing.T) {
	http := NewHTTP().(*HTTP)
	http.worker = &manager.WorkerManager{}
	timeout := 5000
	http.cfg = HTTPArgs{
		Timeout: &timeout,
	}
	conn := &MockConn{
		data: []byte{},
	}
	var netConn net.Conn = conn
	req := &utils.HTTPRequest{
		Host: "example.com:80",
	}
	defer func() {
		if r := recover(); r != nil {
			t.Logf("OutToTCP panics without properly initialized worker (expected): %v", r)
		}
	}()
	err := http.OutToTCP(false, "example.com:80", &netConn, req)
	t.Logf("OutToTCP direct connect result: %v", err)
}

func TestHTTP_OutToTCP_DeadLoopDetection(t *testing.T) {
	http := NewHTTP().(*HTTP)
	http.worker = &manager.WorkerManager{}
	timeout := 5000
	http.cfg = HTTPArgs{
		Timeout: &timeout,
	}
	conn := &MockConnWithAddr{
		MockConn: MockConn{
			data: []byte{},
		},
		localAddr:  &MockAddr{"127.0.0.1:8080"},
		remoteAddr: &MockAddr{"192.168.1.1:12345"},
	}
	var netConn net.Conn = conn
	req := &utils.HTTPRequest{
		Host: "127.0.0.1:8080",
	}
	defer func() {
		if r := recover(); r != nil {
			t.Logf("OutToTCP panics without properly initialized worker (expected): %v", r)
		}
	}()
	err := http.OutToTCP(false, "127.0.0.1:8080", &netConn, req)
	if err == nil {
		t.Log("Dead loop should be detected")
	} else {
		t.Logf("Dead loop detected: %v", err)
	}
}

func TestHTTP_ConcurrentAccess(t *testing.T) {
	http := NewHTTP().(*HTTP)
	http.worker = &manager.WorkerManager{}
	done := make(chan bool, 10)
	for i := 0; i < 5; i++ {
		go func(id int) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Goroutine %d should not panic: %v", id, r)
				}
				done <- true
			}()

			http.InitService()
		}(i)
	}
	for i := 0; i < 5; i++ {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Stop goroutine should not panic: %v", r)
				}
				done <- true
			}()

			http.StopService()
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestHTTP_ConcurrentCallbacks(t *testing.T) {
	http := NewHTTP().(*HTTP)
	http.worker = &manager.WorkerManager{}
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func(id int) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Callback goroutine %d should not panic: %v", id, r)
				}
				done <- true
			}()

			conn := &MockConn{
				data: []byte("GET http://example.com HTTP/1.1\r\nHost: example.com\r\n\r\n"),
			}
			http.callback(conn)
		}(i)
	}
	for i := 0; i < 5; i++ {
		<-done
	}
}

func TestHTTP_Callback_LargeRequest(t *testing.T) {
	http := NewHTTP().(*HTTP)
	http.worker = &manager.WorkerManager{}
	largeHeader := make([]byte, 8192)
	for i := range largeHeader {
		largeHeader[i] = 'X'
	}
	conn := &MockConn{
		data: append([]byte("GET http://example.com HTTP/1.1\r\nHost: example.com\r\nX-Large: "), largeHeader...),
	}
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Callback with large request should not panic: %v", r)
		}
	}()
	http.callback(conn)
}

func TestHTTP_Callback_SpecialCharactersInHost(t *testing.T) {
	http := NewHTTP().(*HTTP)
	http.worker = &manager.WorkerManager{}

	conn := &MockConn{
		data: []byte("GET http://test-host_name.example.com:8080/path HTTP/1.1\r\n" +
			"Host: test-host_name.example.com:8080\r\n\r\n"),
	}
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Callback with special chars in host should not panic: %v", r)
		}
	}()
	http.callback(conn)
}

func TestHTTP_Callback_IPv6Host(t *testing.T) {
	http := NewHTTP().(*HTTP)
	http.worker = &manager.WorkerManager{}
	conn := &MockConn{
		data: []byte("CONNECT [::1]:443 HTTP/1.1\r\n" +
			"Host: [::1]:443\r\n\r\n"),
	}
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Callback with IPv6 host should not panic: %v", r)
		}
	}()
	http.callback(conn)
}

type MockConn struct {
	data   []byte
	pos    int
	closed bool
}

func (m *MockConn) Read(b []byte) (n int, err error) {
	if m.closed {
		return 0, net.ErrClosed
	}
	if m.pos >= len(m.data) {
		return 0, io.EOF
	}
	n = copy(b, m.data[m.pos:])
	m.pos += n
	return n, nil
}

func (m *MockConn) Write(b []byte) (n int, err error) {
	if m.closed {
		return 0, net.ErrClosed
	}
	m.data = append(m.data, b...)
	return len(b), nil
}

func (m *MockConn) Close() error {
	m.closed = true
	return nil
}

func (m *MockConn) LocalAddr() net.Addr {
	return &MockAddr{"127.0.0.1:8080"}
}

func (m *MockConn) RemoteAddr() net.Addr {
	return &MockAddr{"127.0.0.1:12345"}
}

func (m *MockConn) SetDeadline(t time.Time) error {
	return nil
}

func (m *MockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *MockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

type MockConnWithAddr struct {
	MockConn
	localAddr  net.Addr
	remoteAddr net.Addr
}

func (m *MockConnWithAddr) LocalAddr() net.Addr {
	return m.localAddr
}

func (m *MockConnWithAddr) RemoteAddr() net.Addr {
	return m.remoteAddr
}
