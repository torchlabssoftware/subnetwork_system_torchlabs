package manager

import (
	"testing"

	"github.com/google/uuid"
)

func TestUpstreamManager_NewUpstreamManager(t *testing.T) {
	um := NewUpstreamManager()
	if um == nil {
		t.Error("UpstreamManager should not be nil")
	}
	if len(um.upstreams) != 0 {
		t.Errorf("Should start with empty upstreams, got %d", len(um.upstreams))
	}
}

func TestUpstreamManager_SetUpstreams(t *testing.T) {
	um := NewUpstreamManager()
	upstreams := []Upstream{
		createTestUpstream("upstream1", "127.0.0.1", 3128),
		createTestUpstream("upstream2", "127.0.0.2", 3128),
	}
	um.SetUpstreams(upstreams)
	if len(um.upstreams) != 2 {
		t.Errorf("Should have 2 upstreams, got %d", len(um.upstreams))
	}
	if um.upstreams[0].UpstreamTag != "upstream1" {
		t.Errorf("First upstream tag should be upstream1, got %s", um.upstreams[0].UpstreamTag)
	}
	if um.upstreams[1].UpstreamTag != "upstream2" {
		t.Errorf("Second upstream tag should be upstream2, got %s", um.upstreams[1].UpstreamTag)
	}
}

func TestUpstreamManager_SetUpstreams_Empty(t *testing.T) {
	um := NewUpstreamManager()
	upstreams := []Upstream{}
	um.SetUpstreams(upstreams)
	if len(um.upstreams) != 0 {
		t.Errorf("Should have 0 upstreams, got %d", len(um.upstreams))
	}
}

func TestUpstreamManager_HasUpstreams_Empty(t *testing.T) {
	um := NewUpstreamManager()
	if um.HasUpstreams() {
		t.Error("Should not have upstreams when empty")
	}
}

func TestUpstreamManager_HasUpstreams_WithUpstreams(t *testing.T) {
	um := NewUpstreamManager()
	upstreams := []Upstream{
		createTestUpstream("upstream1", "127.0.0.1", 3128),
	}
	um.SetUpstreams(upstreams)
	if !um.HasUpstreams() {
		t.Error("Should have upstreams when set")
	}
}

func TestUpstreamManager_Next_RoundRobin(t *testing.T) {
	um := NewUpstreamManager()
	upstreams := []Upstream{
		createTestUpstream("upstream1", "127.0.0.1", 3128),
		createTestUpstream("upstream2", "127.0.0.2", 3128),
		createTestUpstream("upstream3", "127.0.0.3", 3128),
	}
	um.SetUpstreams(upstreams)
	selected1 := um.Next()
	selected2 := um.Next()
	selected3 := um.Next()
	selected4 := um.Next()
	if selected1.UpstreamTag != "upstream1" {
		t.Errorf("First selection should be upstream1, got %s", selected1.UpstreamTag)
	}
	if selected2.UpstreamTag != "upstream2" {
		t.Errorf("Second selection should be upstream2, got %s", selected2.UpstreamTag)
	}
	if selected3.UpstreamTag != "upstream3" {
		t.Errorf("Third selection should be upstream3, got %s", selected3.UpstreamTag)
	}
	if selected4.UpstreamTag != "upstream1" {
		t.Errorf("Fourth selection should wrap to upstream1, got %s", selected4.UpstreamTag)
	}
}

func TestUpstreamManager_Next_Empty(t *testing.T) {
	um := NewUpstreamManager()
	selected := um.Next()
	if selected != nil {
		t.Error("Should return nil when no upstreams")
	}
}

func TestUpstreamManager_Next_SingleUpstream(t *testing.T) {
	um := NewUpstreamManager()
	upstreams := []Upstream{
		createTestUpstream("upstream1", "127.0.0.1", 3128),
	}
	um.SetUpstreams(upstreams)
	selected1 := um.Next()
	selected2 := um.Next()
	selected3 := um.Next()
	if selected1.UpstreamTag != "upstream1" {
		t.Errorf("Selection should be upstream1, got %s", selected1.UpstreamTag)
	}
	if selected2.UpstreamTag != "upstream1" {
		t.Errorf("Second selection should be upstream1, got %s", selected2.UpstreamTag)
	}
	if selected3.UpstreamTag != "upstream1" {
		t.Errorf("Third selection should be upstream1, got %s", selected3.UpstreamTag)
	}
}

func TestUpstream_GetAddress(t *testing.T) {
	upstream := createTestUpstream("test", "127.0.0.1", 3128)
	expected := "127.0.0.1:3128"
	if upstream.GetAddress() != expected {
		t.Errorf("Address should be %s, got %s", expected, upstream.GetAddress())
	}
}

func TestUpstreamManager_GetUpstreamAddress(t *testing.T) {
	um := NewUpstreamManager()
	upstreams := []Upstream{
		createTestUpstream("upstream1", "127.0.0.1", 3128),
		createTestUpstream("upstream2", "127.0.0.2", 8080),
	}
	um.SetUpstreams(upstreams)
	addresses := um.GetUpstreamAddress()
	if len(addresses) != 2 {
		t.Errorf("Should have 2 addresses, got %d", len(addresses))
	}
	if addresses[0] != "127.0.0.1:3128" {
		t.Errorf("First address should be 127.0.0.1:3128, got %s", addresses[0])
	}
	if addresses[1] != "127.0.0.2:8080" {
		t.Errorf("Second address should be 127.0.0.2:8080, got %s", addresses[1])
	}
}

func TestUpstreamManager_GetUpstreamAddress_Empty(t *testing.T) {
	um := NewUpstreamManager()
	addresses := um.GetUpstreamAddress()
	if len(addresses) != 0 {
		t.Errorf("Should have 0 addresses, got %d", len(addresses))
	}
}

func TestUpstreamManager_ConcurrentAccess(t *testing.T) {
	um := NewUpstreamManager()
	upstreams := []Upstream{
		createTestUpstream("upstream1", "127.0.0.1", 3128),
		createTestUpstream("upstream2", "127.0.0.2", 3128),
	}
	um.SetUpstreams(upstreams)
	done := make(chan bool, 2)
	go func() {
		for i := 0; i < 100; i++ {
			um.Next()
		}
		done <- true
	}()
	go func() {
		for i := 0; i < 10; i++ {
			newUpstreams := []Upstream{
				createTestUpstream("new-upstream1", "127.0.0.1", 3128),
				createTestUpstream("new-upstream2", "127.0.0.2", 3128),
			}
			um.SetUpstreams(newUpstreams)
		}
		done <- true
	}()
	<-done
	<-done
	selected := um.Next()
	if selected == nil {
		t.Error("Should still return upstream after concurrent access")
	}
}

func TestUpstreamManager_AtomicOperations(t *testing.T) {
	um := NewUpstreamManager()
	upstreams := []Upstream{
		createTestUpstream("upstream1", "127.0.0.1", 3128),
		createTestUpstream("upstream2", "127.0.0.2", 3128),
	}
	um.SetUpstreams(upstreams)
	selected1 := um.Next()
	selected2 := um.Next()
	if selected1.UpstreamTag == selected2.UpstreamTag {
		t.Error("Different selections should return different upstreams in round-robin")
	}
}

func createTestUpstream(tag, host string, port int) Upstream {
	return Upstream{
		UpstreamID:       uuid.New(),
		UpstreamTag:      tag,
		UpstreamFormat:   "http",
		UpstreamUsername: "user",
		UpstreamPassword: "pass",
		UpstreamHost:     host,
		UpstreamPort:     port,
		UpstreamProvider: "test",
		Weight:           1,
	}
}
