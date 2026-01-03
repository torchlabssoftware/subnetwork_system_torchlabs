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
