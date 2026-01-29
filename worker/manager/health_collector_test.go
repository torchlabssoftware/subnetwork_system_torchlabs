package manager

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestHealthCollector_NewHealthCollector(t *testing.T) {
	workerID := uuid.New()
	hc := NewHealthCollector(workerID)
	if hc == nil {
		t.Error("HealthCollector should not be nil")
	}
	if hc.workerID != workerID {
		t.Errorf("WorkerID should be set, expected %s, got %s", workerID, hc.workerID)
	}
	if len(hc.samples) != 0 {
		t.Errorf("Should start with empty samples, got %d", len(hc.samples))
	}
	if len(hc.upstreamStats) != 0 {
		t.Errorf("Should start with empty upstream stats, got %d", len(hc.upstreamStats))
	}
}

func TestHealthCollector_RecordSample(t *testing.T) {
	workerID := uuid.New()
	hc := NewHealthCollector(workerID)
	hc.RecordSample()
	if len(hc.samples) != 1 {
		t.Errorf("Should have 1 sample, got %d", len(hc.samples))
	}
	sample := hc.samples[0]
	if sample.Timestamp.IsZero() {
		t.Error("Sample timestamp should be set")
	}
	if sample.CpuUsage < 0 || sample.CpuUsage > 100 {
		t.Errorf("CPU usage should be between 0 and 100, got %f", sample.CpuUsage)
	}
	if sample.MemoryUsage < 0 || sample.MemoryUsage > 100 {
		t.Errorf("Memory usage should be between 0 and 100, got %f", sample.MemoryUsage)
	}
}

func TestHealthCollector_RecordSample_Multiple(t *testing.T) {
	workerID := uuid.New()
	hc := NewHealthCollector(workerID)
	for i := 0; i < 5; i++ {
		hc.RecordSample()
		time.Sleep(10 * time.Millisecond)
	}
	if len(hc.samples) != 5 {
		t.Errorf("Should have 5 samples, got %d", len(hc.samples))
	}
}

func TestHealthCollector_ConnectionTracking(t *testing.T) {
	workerID := uuid.New()
	hc := NewHealthCollector(workerID)
	hc.IncrementConnection()
	active := atomic.LoadUint32(&hc.activeConnections)
	if active != 1 {
		t.Errorf("Active connections should be 1, got %d", active)
	}
	for i := 0; i < 10; i++ {
		hc.IncrementConnection()
	}
	active = atomic.LoadUint32(&hc.activeConnections)
	if active != 11 {
		t.Errorf("Active connections should be 11, got %d", active)
	}
	hc.DecrementConnection()
	active = atomic.LoadUint32(&hc.activeConnections)
	if active != 10 {
		t.Errorf("Active connections should be 10 after decrement, got %d", active)
	}
	total := atomic.LoadUint64(&hc.totalConnections)
	if total == 0 {
		t.Error("Total connections should be greater than 0")
	}
}

func TestHealthCollector_ThroughputTracking(t *testing.T) {
	workerID := uuid.New()
	hc := NewHealthCollector(workerID)
	hc.AddThroughput(1024)
	throughput := atomic.LoadUint64(&hc.bytesThroughput)
	if throughput != 1024 {
		t.Errorf("Throughput should be 1024, got %d", throughput)
	}
	hc.AddThroughput(2048)
	throughput = atomic.LoadUint64(&hc.bytesThroughput)
	if throughput != 3072 {
		t.Errorf("Throughput should be 3072, got %d", throughput)
	}
}

func TestHealthCollector_ErrorSuccessTracking(t *testing.T) {
	workerID := uuid.New()
	hc := NewHealthCollector(workerID)
	hc.RecordSuccess()
	successes := atomic.LoadUint64(&hc.successCount)
	if successes != 1 {
		t.Errorf("Success count should be 1, got %d", successes)
	}
	hc.RecordError()
	errors := atomic.LoadUint64(&hc.errorCount)
	if errors != 1 {
		t.Errorf("Error count should be 1, got %d", errors)
	}
	for i := 0; i < 5; i++ {
		hc.RecordSuccess()
		hc.RecordError()
	}
	successes = atomic.LoadUint64(&hc.successCount)
	errors = atomic.LoadUint64(&hc.errorCount)
	if successes != 6 {
		t.Errorf("Success count should be 6, got %d", successes)
	}
	if errors != 6 {
		t.Errorf("Error count should be 6, got %d", errors)
	}
}

func TestHealthCollector_UpstreamLatency(t *testing.T) {
	workerID := uuid.New()
	hc := NewHealthCollector(workerID)
	upstreamID := uuid.New()
	hc.RecordUpstreamLatency(upstreamID, "test-upstream", 100*time.Millisecond, false)
	hc.upstreamMu.RLock()
	stats, exists := hc.upstreamStats[upstreamID]
	hc.upstreamMu.RUnlock()
	if !exists {
		t.Error("Upstream stats should exist")
	}
	if stats.RequestCount != 1 {
		t.Errorf("Request count should be 1, got %d", stats.RequestCount)
	}
	if stats.TotalLatency != 100 {
		t.Errorf("Total latency should be 100, got %d", stats.TotalLatency)
	}
	if stats.ErrorCount != 0 {
		t.Errorf("Error count should be 0, got %d", stats.ErrorCount)
	}
	hc.RecordUpstreamLatency(upstreamID, "test-upstream", 50*time.Millisecond, true)
	hc.upstreamMu.RLock()
	stats = hc.upstreamStats[upstreamID]
	hc.upstreamMu.RUnlock()
	if stats.RequestCount != 2 {
		t.Errorf("Request count should be 2, got %d", stats.RequestCount)
	}
	if stats.TotalLatency != 150 {
		t.Errorf("Total latency should be 150, got %d", stats.TotalLatency)
	}
	if stats.ErrorCount != 1 {
		t.Errorf("Error count should be 1, got %d", stats.ErrorCount)
	}
}

func TestHealthCollector_UpdateWorkerInfo(t *testing.T) {
	workerID := uuid.New()
	hc := NewHealthCollector(workerID)
	hc.UpdateWorkerInfo("test-worker", "us-east-1")
	if hc.workerName != "test-worker" {
		t.Errorf("Worker name should be test-worker, got %s", hc.workerName)
	}
	if hc.region != "us-east-1" {
		t.Errorf("Region should be us-east-1, got %s", hc.region)
	}
}

func TestHealthCollector_BuildWorkerHealth(t *testing.T) {
	workerID := uuid.New()
	hc := NewHealthCollector(workerID)
	hc.UpdateWorkerInfo("test-worker", "us-east-1")
	hc.IncrementConnection()
	hc.IncrementConnection()
	hc.AddThroughput(102400)
	hc.RecordSuccess()
	hc.RecordSuccess()
	hc.RecordError()
	for i := 0; i < 3; i++ {
		hc.RecordSample()
	}
	upstreamID := uuid.New()
	hc.RecordUpstreamLatency(upstreamID, "test-upstream", 100*time.Millisecond, false)
	hc.RecordUpstreamLatency(upstreamID, "test-upstream", 200*time.Millisecond, false)
	health := hc.BuildWorkerHealth()
	if health.WorkerID != workerID {
		t.Errorf("Worker ID should match, expected %s, got %s", workerID, health.WorkerID)
	}
	if health.WorkerName != "test-worker" {
		t.Errorf("Worker name should be test-worker, got %s", health.WorkerName)
	}
	if health.Region != "us-east-1" {
		t.Errorf("Region should be us-east-1, got %s", health.Region)
	}
	if health.ActiveConnections != 2 {
		t.Errorf("Active connections should be 2, got %d", health.ActiveConnections)
	}
	if health.TotalConnections != 2 {
		t.Errorf("Total connections should be 2, got %d", health.TotalConnections)
	}
	if health.BytesThroughputPerSec == 0 {
		t.Error("Bytes throughput should be greater than 0")
	}
	expectedErrorRate := float32(1) / float32(3) * 100
	if health.ErrorRate != expectedErrorRate {
		t.Errorf("Error rate should be %f, got %f", expectedErrorRate, health.ErrorRate)
	}
	if health.Status == "" {
		t.Error("Status should be set")
	}
	if len(health.Upstreams) != 1 {
		t.Errorf("Should have 1 upstream health, got %d", len(health.Upstreams))
	}
	upstreamHealth := health.Upstreams[0]
	if upstreamHealth.UpstreamID != upstreamID {
		t.Errorf("Upstream ID should match")
	}
	if upstreamHealth.UpstreamTag != "test-upstream" {
		t.Errorf("Upstream tag should be test-upstream, got %s", upstreamHealth.UpstreamTag)
	}
	if upstreamHealth.Latency != 150 {
		t.Errorf("Upstream latency should be 150, got %d", upstreamHealth.Latency)
	}
	if upstreamHealth.ErrorRate != 0 {
		t.Errorf("Upstream error rate should be 0, got %f", upstreamHealth.ErrorRate)
	}
}

func TestHealthCollector_BuildWorkerHealth_Reset(t *testing.T) {
	workerID := uuid.New()
	hc := NewHealthCollector(workerID)
	hc.IncrementConnection()
	hc.AddThroughput(102400)
	hc.RecordSuccess()
	hc.RecordSample()
	health := hc.BuildWorkerHealth()
	if health.TotalConnections != 1 {
		t.Errorf("Total connections should be 1, got %d", health.TotalConnections)
	}
	if health.BytesThroughputPerSec == 0 {
		t.Error("Bytes throughput should be set")
	}
	health2 := hc.BuildWorkerHealth()
	if health2.TotalConnections != 0 {
		t.Errorf("Total connections should be 0 after reset, got %d", health2.TotalConnections)
	}
	if health2.BytesThroughputPerSec != 0 {
		t.Errorf("Bytes throughput should be 0 after reset, got %d", health2.BytesThroughputPerSec)
	}
}

func TestHealthCollector_StatusDetermination(t *testing.T) {
	workerID := uuid.New()
	t.Run("Healthy", func(t *testing.T) {
		hc := NewHealthCollector(workerID)
		hc.IncrementConnection()
		hc.RecordSuccess()
		hc.RecordSuccess()
		hc.RecordSuccess()
		health := hc.BuildWorkerHealth()
		if health.Status != "healthy" {
			t.Errorf("Status should be healthy, got %s", health.Status)
		}
	})
	t.Run("Degraded", func(t *testing.T) {
		hc := NewHealthCollector(workerID)
		hc.IncrementConnection()
		for i := 0; i < 10; i++ {
			hc.RecordError()
		}
		hc.RecordSuccess()
		health := hc.BuildWorkerHealth()
		if health.Status != "degraded" {
			t.Errorf("Status should be degraded, got %s", health.Status)
		}
	})
	t.Run("Idle", func(t *testing.T) {
		hc := NewHealthCollector(workerID)
		health := hc.BuildWorkerHealth()
		if health.Status != "idle" {
			t.Errorf("Status should be idle, got %s", health.Status)
		}
	})
}

func TestHealthCollector_StartStop(t *testing.T) {
	workerID := uuid.New()
	hc := NewHealthCollector(workerID)
	hc.Start()
	time.Sleep(100 * time.Millisecond)
	hc.Stop()
}

func TestHealthCollector_ConcurrentAccess(t *testing.T) {
	workerID := uuid.New()
	hc := NewHealthCollector(workerID)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			hc.RecordSample()
			time.Sleep(time.Millisecond)
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			hc.IncrementConnection()
			hc.DecrementConnection()
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			hc.AddThroughput(1024)
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			hc.RecordSuccess()
			hc.RecordError()
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			hc.BuildWorkerHealth()
			time.Sleep(10 * time.Millisecond)
		}
	}()
	wg.Wait()
	health := hc.BuildWorkerHealth()
	if health.WorkerID != workerID {
		t.Error("Should still work after concurrent access")
	}
}
