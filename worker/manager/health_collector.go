package manager

import (
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

type HealthSample struct {
	CpuUsage    float32
	MemoryUsage float32
	Timestamp   time.Time
}

type UpstreamStats struct {
	UpstreamID   uuid.UUID
	UpstreamTag  string
	TotalLatency int64
	RequestCount uint64
	ErrorCount   uint64
}

type HealthCollector struct {
	workerID   uuid.UUID
	workerName string
	region     string

	samples []HealthSample
	mu      sync.Mutex

	activeConnections uint32
	totalConnections  uint64
	bytesThroughput   uint64
	errorCount        uint64
	successCount      uint64

	upstreamStats map[uuid.UUID]*UpstreamStats
	upstreamMu    sync.RWMutex

	sampleTicker *time.Ticker
	stopCh       chan struct{}
}

func NewHealthCollector(workerID uuid.UUID) *HealthCollector {
	hc := &HealthCollector{
		workerID:      workerID,
		samples:       make([]HealthSample, 0),
		upstreamStats: make(map[uuid.UUID]*UpstreamStats),

		stopCh: make(chan struct{}),
	}
	return hc
}

func (h *HealthCollector) Start() {
	h.sampleTicker = time.NewTicker(1 * time.Minute)
	go func() {
		for {
			select {
			case <-h.sampleTicker.C:
				h.RecordSample()
			case <-h.stopCh:
				return
			}
		}
	}()
	log.Println("[HealthCollector] Started periodic sampling")
}

func (h *HealthCollector) Stop() {
	if h.sampleTicker != nil {
		h.sampleTicker.Stop()
	}
	close(h.stopCh)
	log.Println("[HealthCollector] Stopped")
}

func (h *HealthCollector) RecordSample() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	memUsage := float32(0)
	if memStats.Sys > 0 {
		memUsage = float32(memStats.Alloc) / float32(memStats.Sys) * 100
	}

	numGoroutines := runtime.NumGoroutine()
	numCPU := runtime.NumCPU()
	cpuUsage := float32(numGoroutines) / float32(numCPU*100) * 100
	if cpuUsage > 100 {
		cpuUsage = 100
	}

	sample := HealthSample{
		CpuUsage:    cpuUsage,
		MemoryUsage: memUsage,
		Timestamp:   time.Now(),
	}

	h.mu.Lock()
	h.samples = append(h.samples, sample)
	h.mu.Unlock()
}

func (h *HealthCollector) IncrementConnection() {
	atomic.AddUint32(&h.activeConnections, 1)
	atomic.AddUint64(&h.totalConnections, 1)
}

func (h *HealthCollector) DecrementConnection() {
	atomic.AddUint32(&h.activeConnections, ^uint32(0))
}

func (h *HealthCollector) AddThroughput(bytes uint64) {
	atomic.AddUint64(&h.bytesThroughput, bytes)
}

func (h *HealthCollector) RecordError() {
	atomic.AddUint64(&h.errorCount, 1)
}

func (h *HealthCollector) RecordSuccess() {
	atomic.AddUint64(&h.successCount, 1)
}

func (h *HealthCollector) RecordUpstreamLatency(upstreamID uuid.UUID, upstreamTag string, latency time.Duration, isError bool) {
	h.upstreamMu.Lock()
	defer h.upstreamMu.Unlock()

	stats, exists := h.upstreamStats[upstreamID]
	if !exists {
		stats = &UpstreamStats{
			UpstreamID:  upstreamID,
			UpstreamTag: upstreamTag,
		}
		h.upstreamStats[upstreamID] = stats
	}

	stats.TotalLatency += latency.Milliseconds()
	stats.RequestCount++
	if isError {
		stats.ErrorCount++
	}
}

func (h *HealthCollector) UpdateWorkerInfo(workerName, region string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.workerName = workerName
	h.region = region
}

func (h *HealthCollector) BuildWorkerHealth() WorkerHealth {
	h.mu.Lock()
	samples := h.samples
	h.samples = make([]HealthSample, 0)
	h.mu.Unlock()

	var avgCpu, avgMem float32
	if len(samples) > 0 {
		var totalCpu, totalMem float32
		for _, s := range samples {
			totalCpu += s.CpuUsage
			totalMem += s.MemoryUsage
		}
		avgCpu = totalCpu / float32(len(samples))
		avgMem = totalMem / float32(len(samples))
	}

	activeConns := atomic.LoadUint32(&h.activeConnections)
	totalConns := atomic.SwapUint64(&h.totalConnections, 0)
	throughput := atomic.SwapUint64(&h.bytesThroughput, 0)
	errors := atomic.SwapUint64(&h.errorCount, 0)
	successes := atomic.SwapUint64(&h.successCount, 0)

	var errorRate float32
	totalRequests := errors + successes
	if totalRequests > 0 {
		errorRate = float32(errors) / float32(totalRequests) * 100
	}

	bytesPerSec := throughput / 3600

	status := "healthy"
	if errorRate > 50 {
		status = "degraded"
	}
	if activeConns == 0 && totalConns == 0 {
		status = "idle"
	}

	h.upstreamMu.Lock()
	upstreams := make([]UpstreamHealth, 0, len(h.upstreamStats))
	for _, stats := range h.upstreamStats {
		var avgLatency int64
		var upstreamErrorRate float32
		if stats.RequestCount > 0 {
			avgLatency = stats.TotalLatency / int64(stats.RequestCount)
			upstreamErrorRate = float32(stats.ErrorCount) / float32(stats.RequestCount) * 100
		}

		upstreamStatus := "healthy"
		if upstreamErrorRate > 80 {
			upstreamStatus = "unhealthy"
		} else if upstreamErrorRate > 50 {
			upstreamStatus = "degraded"
		}

		upstreams = append(upstreams, UpstreamHealth{
			UpstreamID:  stats.UpstreamID,
			UpstreamTag: stats.UpstreamTag,
			Status:      upstreamStatus,
			Latency:     avgLatency,
			ErrorRate:   upstreamErrorRate,
		})
	}

	h.upstreamStats = make(map[uuid.UUID]*UpstreamStats)
	h.upstreamMu.Unlock()

	return WorkerHealth{
		WorkerID:              h.workerID,
		WorkerName:            h.workerName,
		Region:                h.region,
		Status:                status,
		CpuUsage:              avgCpu,
		MemoryUsage:           avgMem,
		ActiveConnections:     activeConns,
		TotalConnections:      totalConns,
		BytesThroughputPerSec: bytesPerSec,
		ErrorRate:             errorRate,
		Upstreams:             upstreams,
	}
}
