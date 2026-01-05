package manager

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
)

type Pool struct {
	PoolId  uuid.UUID
	PoolTag string
	//get region from captain
	Region        string
	PoolPort      int
	PoolSubdomain string
	Upstreams     []Upstream
}

type Upstream struct {
	UpstreamID       uuid.UUID
	UpstreamTag      string
	UpstreamFormat   string
	UpstreamUsername string
	UpstreamPassword string
	UpstreamHost     string
	UpstreamPort     int
	UpstreamProvider string
	Weight           int
}

func NewPool(poolId uuid.UUID, poolTag string, poolPort int, poolSubdomain string, upstreams []Upstream) *Pool {
	pool := &Pool{
		PoolId:        poolId,
		PoolTag:       poolTag,
		PoolPort:      poolPort,
		PoolSubdomain: poolSubdomain,
		Upstreams:     upstreams,
		Region:        "",
	}
	return pool
}

// UpstreamManager handles round-robin selection of upstream proxies
type UpstreamManager struct {
	upstreams []Upstream
	index     uint64 // atomic counter for round-robin
	mu        sync.RWMutex
}

// NewUpstreamManager creates a new upstream manager
func NewUpstreamManager() *UpstreamManager {
	return &UpstreamManager{
		upstreams: make([]Upstream, 0),
	}
}

// SetUpstreams updates the list of available upstreams (called when Captain sends config)
func (m *UpstreamManager) SetUpstreams(upstreams []Upstream) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.upstreams = upstreams
	log.Printf("[UpstreamManager] Updated upstreams, count: %d", len(upstreams))
	for i, u := range upstreams {
		log.Printf("[UpstreamManager] Upstream %d: %s:%d (tag: %s)", i, u.UpstreamHost, u.UpstreamPort, u.UpstreamTag)
	}
}

// Next returns the next upstream using round-robin selection
// Returns nil if no upstreams are configured
func (m *UpstreamManager) Next() *Upstream {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.upstreams) == 0 {
		return nil
	}

	// Atomic increment and get index
	idx := atomic.AddUint64(&m.index, 1) - 1
	selectedIdx := idx % uint64(len(m.upstreams))

	upstream := m.upstreams[selectedIdx]
	log.Printf("[UpstreamManager] Round-robin selected upstream %d: %s:%d", selectedIdx, upstream.UpstreamHost, upstream.UpstreamPort)

	return &upstream
}

// HasUpstreams returns true if there are upstreams configured
func (m *UpstreamManager) HasUpstreams() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.upstreams) > 0
}

// Count returns the number of configured upstreams
func (m *UpstreamManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.upstreams)
}

// GetAddress returns the host:port string for an upstream
func (u *Upstream) GetAddress() string {
	return fmt.Sprintf("%s:%d", u.UpstreamHost, u.UpstreamPort)
}
