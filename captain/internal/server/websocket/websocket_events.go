package server

import (
	"github.com/google/uuid"
)

type Event struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type EventHandler func(event Event, w *Worker) error

type loginPayload struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type replyPayload struct {
	Success bool        `json:"success"`
	Payload interface{} `json:"payload"`
}

type loginSuccessPayload struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	Password    string    `json:"password"`
	Status      string    `json:"status"`
	IpWhitelist []string  `json:"ip_whitelist"`
	Pools       []string  `json:"pools"`
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

type ConfigPayload struct {
	WorkerName    string           `json:"worker_name"`
	Region        string           `json:"region"`
	PoolID        uuid.UUID        `json:"pool_id"`
	PoolTag       string           `json:"pool_tag"`
	PoolPort      int              `json:"pool_port"`
	PoolSubdomain string           `json:"pool_subdomain"`
	Upstreams     []UpstreamConfig `json:"upstreams"`
}
