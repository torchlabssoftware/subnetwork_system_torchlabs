package manager

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
)

type Pool struct {
	PoolId        uuid.UUID
	PoolTag       string
	Region        string
	PoolPort      int
	PoolSubdomain string
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

func NewPool(poolId uuid.UUID, poolTag string, poolPort int, poolSubdomain string) *Pool {
	pool := &Pool{
		PoolId:        poolId,
		PoolTag:       poolTag,
		PoolPort:      poolPort,
		PoolSubdomain: poolSubdomain,
		Region:        "",
	}
	return pool
}

type UpstreamManager struct {
	upstreams []Upstream
	index     uint64
	mu        sync.RWMutex
}

func NewUpstreamManager() *UpstreamManager {
	return &UpstreamManager{
		upstreams: make([]Upstream, 0),
	}
}

func (m *UpstreamManager) SetUpstreams(upstreams []Upstream) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.upstreams = upstreams
	log.Printf("[UpstreamManager] Updated upstreams, count: %d", len(upstreams))
	for i, u := range upstreams {
		log.Printf("[UpstreamManager] Upstream %d: %s:%d (tag: %s)", i, u.UpstreamHost, u.UpstreamPort, u.UpstreamTag)
	}
}

func (m *UpstreamManager) Next() *Upstream {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.upstreams) == 0 {
		return nil
	}
	idx := atomic.AddUint64(&m.index, 1) - 1
	selectedIdx := idx % uint64(len(m.upstreams))
	upstream := m.upstreams[selectedIdx]
	log.Printf("[UpstreamManager] Round-robin selected upstream %d: %s:%d", selectedIdx, upstream.UpstreamHost, upstream.UpstreamPort)
	return &upstream
}

func (m *UpstreamManager) HasUpstreams() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.upstreams) > 0
}

func (u *Upstream) GetAddress() string {
	return fmt.Sprintf("%s:%d", u.UpstreamHost, u.UpstreamPort)
}

func (u *UpstreamManager) GetUpstreamAddress() []string {
	u.mu.RLock()
	defer u.mu.RUnlock()
	addresses := make([]string, len(u.upstreams))
	for i, u := range u.upstreams {
		addresses[i] = u.GetAddress()
	}
	return addresses
}
