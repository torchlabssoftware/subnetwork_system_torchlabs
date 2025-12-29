package websocket

import (
	"github.com/google/uuid"
)

type Event struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type LoginRequest struct {
	WorkerID string `json:"worker_id"`
}

type LoginResponse struct {
	Otp string `json:"otp"`
}

type ConfigPayload struct {
	PoolID        string           `json:"pool_id"`
	PoolTag       string           `json:"pool_tag"`
	PoolPort      int32            `json:"pool_port"`
	PoolSubdomain string           `json:"pool_subdomain"`
	Upstreams     []UpstreamConfig `json:"upstreams"`
}

type UpstreamConfig struct {
	UpstreamID      string `json:"upstream_id"`
	UpstreamTag     string `json:"upstream_tag"`
	UpstreamAddress string `json:"upstream_address"`
	UpstreamHost    string `json:"upstream_host"`
	UpstreamPort    int32  `json:"upstream_port"`
	Weight          int32  `json:"weight"`
}

type User struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	Password    string    `json:"password"`
	Status      string    `json:"status"`
	IpWhitelist []string  `json:"ip_whitelist"`
	Pools       []string  `json:"pools"`
}
