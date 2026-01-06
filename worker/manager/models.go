package manager

import (
	"github.com/google/uuid"
)

type Event struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type EventHandler func(event Event) error

type ErrorPayload struct {
	Message string      `json:"message"`
	Payload interface{} `json:"payload"`
}

type LoginRequest struct {
	WorkerID string `json:"worker_id"`
}

type LoginResponse struct {
	Otp string `json:"otp"`
}

type ConfigPayload struct {
	WorkerName    string           `json:"worker_name"`
	PoolID        uuid.UUID        `json:"pool_id"`
	PoolTag       string           `json:"pool_tag"`
	PoolPort      int              `json:"pool_port"`
	PoolSubdomain string           `json:"pool_subdomain"`
	Upstreams     []UpstreamConfig `json:"upstreams"`
}

type UpstreamConfig struct {
	UpstreamID       uuid.UUID `json:"upstream_id"`
	UpstreamTag      string    `json:"upstream_tag"`
	UpstreamFormat   string    `json:"upstream_format"`
	UpstreamUsername string    `json:"upstream_username"`
	UpstreamPassword string    `json:"upstream_password"`
	UpstreamHost     string    `json:"upstream_host"`
	UpstreamPort     int       `json:"upstream_port"`
	UpstreamProvider string    `json:"upstream_provider"`
	Weight           int       `json:"weight"`
}

type UserPayload struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	Password    string    `json:"password"`
	Status      string    `json:"status"`
	IpWhitelist []string  `json:"ip_whitelist"`
	Pools       []string  `json:"pools"`
}

type User struct {
	ID          uuid.UUID
	Status      string
	IpWhitelist []string
	Pools       []string
}

// UserDataUsage tracks per-user data usage for reporting to Captain
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

// WorkerHealth represents the health status of a worker for telemetry
type WorkerHealth struct {
	WorkerID              uuid.UUID        `json:"worker_id"`
	WorkerName            string           `json:"worker_name"`
	Region                string           `json:"region"`
	Status                string           `json:"status"`
	CpuUsage              float32          `json:"cpu_usage"`
	MemoryUsage           float32          `json:"memory_usage"`
	ActiveConnections     uint32           `json:"active_connections"`
	TotalConnections      uint64           `json:"total_connections"`
	BytesThroughputPerSec uint64           `json:"bytes_throughput_per_sec"`
	ErrorRate             float32          `json:"error_rate"`
	Upstreams             []UpstreamHealth `json:"upstreams"`
}

// UpstreamHealth represents the health status of an upstream proxy
type UpstreamHealth struct {
	UpstreamID  uuid.UUID `json:"upstream_id"`
	UpstreamTag string    `json:"upstream_tag"`
	Status      string    `json:"status"`
	Latency     int64     `json:"latency"`
	ErrorRate   float32   `json:"error_rate"`
}
