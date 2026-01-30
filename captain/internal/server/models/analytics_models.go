package server

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type UserDataUsage struct {
	UserID          uuid.UUID `json:"user_id"`
	Username        string    `json:"username"`
	PoolID          uuid.UUID `json:"pool_id"`
	PoolName        string    `json:"pool_name"`
	WorkerID        uuid.UUID `json:"worker_id"`
	WorkerRegion    string    `json:"worker_region"`
	BytesSent       uint64    `json:"bytes_sent"`
	BytesReceived   uint64    `json:"bytes_received"`
	SourceIP        string    `json:"source_ip"`
	Protocol        string    `json:"protocol"`
	DestinationHost string    `json:"destination_host"`
	DestinationPort uint16    `json:"destination_port"`
	StatusCode      uint16    `json:"status_code"`
}

type UpstreamHealth struct {
	UpstreamID  uuid.UUID `json:"upstream_id"`
	UpstreamTag string    `json:"upstream_tag"`
	Status      string    `json:"status"`
	Latency     int64     `json:"latency"`
	ErrorRate   float32   `json:"error_rate"`
}

type WorkerHealth struct {
	WorkerID              uuid.UUID        `json:"worker_id"`
	WorkerName            string           `json:"worker_name"`
	Region                string           `json:"region"`
	PoolTag               string           `json:"pool_tag"`
	Status                string           `json:"status"`
	CpuUsage              float32          `json:"cpu_usage"`
	MemoryUsage           float32          `json:"memory_usage"`
	ActiveConnections     uint32           `json:"active_connections"`
	TotalConnections      uint64           `json:"total_connections"`
	BytesThroughputPerSec uint64           `json:"bytes_throughput_per_sec"`
	ErrorRate             float32          `json:"error_rate"`
	Upstreams             []UpstreamHealth `json:"upstreams"`
}

type WebsiteAccess struct {
	UserID        uuid.UUID `json:"user_id"`
	Username      string    `json:"username"`
	Domain        string    `json:"domain"`
	Subdomain     string    `json:"subdomain"`
	FullURL       string    `json:"full_url"`
	BytesSent     uint64    `json:"bytes_sent"`
	BytesReceived uint64    `json:"bytes_received"`
	RequestMethod string    `json:"request_method"`
	StatusCode    uint16    `json:"status_code"`
	ContentType   string    `json:"content_type"`

	SourceIP string `json:"source_ip"`
}

type UserUsageHourly struct {
	Date               time.Time `ch:"date" json:"date"`
	Hour               time.Time `ch:"hour" json:"hour"`
	UserID             uuid.UUID `ch:"user_id" json:"user_id"`
	Username           string    `ch:"username" json:"username"`
	BytesSent          uint64    `ch:"bytes_sent" json:"bytes_sent"`
	BytesReceived      uint64    `ch:"bytes_received" json:"bytes_received"`
	RequestCount       uint64    `ch:"request_count" json:"request_count"`
	UniqueSessions     uint64    `ch:"unique_sessions" json:"unique_sessions"`
	UniqueDestinations uint64    `ch:"unique_destinations" json:"unique_destinations"`
}

type AnalyticsService interface {
	RecordUserDataUsage(ctx context.Context, data UserDataUsage) error
	RecordWorkerHealth(ctx context.Context, data WorkerHealth) error
	RecordWebsiteAccess(ctx context.Context, data WebsiteAccess) error
	GetUserUsage(ctx context.Context, userID uuid.UUID, from, to time.Time, granularity string) (interface{}, error)
	GetWorkerHealth(ctx context.Context, workerID uuid.UUID, from, to time.Time) ([]WorkerHealth, error)
	GetUserWebsiteAccess(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]WebsiteAccess, error)
	StartWorkers()
}
