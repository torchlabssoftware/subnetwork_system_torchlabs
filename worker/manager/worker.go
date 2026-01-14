package manager

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

type worker struct {
	ID         uuid.UUID
	Name       string
	Region     string
	Pool       *Pool
	CaptainURL string
	APIKey     string
}

type WorkerManager struct {
	Worker           worker
	websocketManager *WebsocketManager
	upstreamManager  *UpstreamManager
	HealthCollector  *HealthCollector
	userManager      *UserManager
}

func NewWorkerManager(workerID, baseURL, apiKey string) (*WorkerManager, error) {
	workerUUID, err := uuid.Parse(workerID)
	if err != nil {
		return nil, fmt.Errorf("[worker] invalid worker ID: %w", err)
	}
	upstreamManager := NewUpstreamManager()
	healthCollector := NewHealthCollector(workerUUID, "", "", upstreamManager)
	userManager := NewUserManager()
	w := &WorkerManager{
		Worker: worker{
			ID:         workerUUID,
			CaptainURL: baseURL,
			APIKey:     apiKey,
		},
		upstreamManager: upstreamManager,
		HealthCollector: healthCollector,
		userManager:     userManager,
	}
	return w, nil
}

func (c *WorkerManager) Start() {
	c.HealthCollector.Start()
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			c.SendHealthTelemetry()
		}
	}()
	go func() {
		for {
			if err := c.Connect(); err != nil {
				log.Printf("[worker] Connection failed: %v. Retrying in 5s...", err)
				time.Sleep(5 * time.Second)
				continue
			}
			log.Println("[worker] Disconnected. Reconnecting in 2s...")
			time.Sleep(2 * time.Second)
		}
	}()
}

func (c *WorkerManager) Connect() error {
	otp, err := c.login()
	if err != nil {
		return fmt.Errorf("[worker] captain login failed: %v", err)
	}
	conn, err := ConnnectToWebsocket(c.Worker.CaptainURL, c.Worker.APIKey, otp)
	if err != nil {
		return fmt.Errorf("[worker] connect to websocket failed: %v", err)
	}
	c.websocketManager = NewWebsocketManager(c, conn)
	defer func() {
		c.websocketManager.Stop()
	}()
	c.websocketManager.Start()
	return fmt.Errorf("[worker] connection closed")
}

func (c *WorkerManager) login() (string, error) {
	return LogintoCaptain(c.Worker.CaptainURL, c.Worker.ID.String(), c.Worker.APIKey)
}

func (c *WorkerManager) processConfig(cfg ConfigPayload) {
	c.Worker.Name = cfg.WorkerName
	c.Worker.Region = cfg.Region
	c.Worker.Pool = NewPool(cfg.PoolID, cfg.PoolTag, cfg.PoolPort, cfg.PoolSubdomain)
	upstreams := make([]Upstream, 0)
	for _, upstream := range cfg.Upstreams {
		upstreams = append(upstreams, Upstream{
			UpstreamID:       upstream.UpstreamID,
			UpstreamTag:      upstream.UpstreamTag,
			UpstreamFormat:   upstream.UpstreamFormat,
			UpstreamUsername: upstream.UpstreamUsername,
			UpstreamPassword: upstream.UpstreamPassword,
			UpstreamHost:     upstream.UpstreamHost,
			UpstreamPort:     int(upstream.UpstreamPort),
			UpstreamProvider: upstream.UpstreamProvider,
			Weight:           upstream.Weight,
		})
	}
	c.upstreamManager.SetUpstreams(upstreams)
	c.HealthCollector.UpdateWorkerInfo(cfg.WorkerName, c.Worker.Pool.Region)
	log.Printf("[worker] Configuration received for Pool: %s", cfg.PoolTag)
	log.Printf("[worker] Upstreams count: %d", len(cfg.Upstreams))
}

func (c *WorkerManager) VerifyUser(user, pass string) bool {
	return c.userManager.VerifyUser(user, pass, func(event Event) {
		c.websocketManager.WriteEvent(event)
	}, c.Worker.Pool.PoolTag)
}

func (c *WorkerManager) processVerifyUserResponse(userPayload UserPayload) {
	c.userManager.processVerifyUserResponse(userPayload)
}

func (c *WorkerManager) GetUser(user string) string {
	if user, ok := c.userManager.GetUser(user); ok {
		return user.Username
	}
	return ""
}

func (c *WorkerManager) HasUpstreams() bool {
	return c.upstreamManager != nil && c.upstreamManager.HasUpstreams()
}

func (c *WorkerManager) NextUpstream(username, session string) *Upstream {
	if session == "" {
		return c.upstreamManager.Next()
	}
	if user, ok := c.userManager.GetUser(username); ok {
		sessions := user.Sessions
		if upstream, ok := sessions[session]; ok {
			log.Println("[worker] Using existing upstream for session:", session)
			return &upstream
		}
		upstream := c.upstreamManager.Next()
		sessions[session] = *upstream
		return upstream
	}
	return c.upstreamManager.Next()
}

func (c *WorkerManager) RecordUpstreamLatency(upstream *Upstream, connectLatency time.Duration, err error) {
	c.HealthCollector.RecordUpstreamLatency(
		upstream.UpstreamID,
		upstream.UpstreamTag,
		connectLatency,
		err != nil,
	)
}

// SendDataUsage sends a user data usage event to Captain via WebSocket
func (c *WorkerManager) SendDataUsage(usage UserDataUsage) {
	if c.websocketManager == nil {
		log.Printf("[DataUsage] WebSocket not connected, cannot send data usage")
		return
	}

	event := Event{
		Type:    "telemetry_usage",
		Payload: usage,
	}

	c.websocketManager.WriteEvent(event)
	log.Printf("[DataUsage] Sent usage: user=%s, bytes_sent=%d, bytes_received=%d, dest=%s:%d",
		usage.Username, usage.BytesSent, usage.BytesReceived, usage.DestinationHost, usage.DestinationPort)
}

// SendHealthTelemetry sends worker health telemetry to Captain via WebSocket
func (c *WorkerManager) SendHealthTelemetry() {
	if c.websocketManager == nil {
		log.Printf("[HealthTelemetry] WebSocket not connected, cannot send health telemetry")
		return
	}

	health := c.HealthCollector.BuildWorkerHealth()
	event := Event{
		Type:    "telemetry_health",
		Payload: health,
	}

	c.websocketManager.WriteEvent(event)
	log.Printf("[HealthTelemetry] Sent health: status=%s, cpu=%.2f%%, mem=%.2f%%, active_conns=%d, throughput=%d bytes/sec",
		health.Status, health.CpuUsage, health.MemoryUsage, health.ActiveConnections, health.BytesThroughputPerSec)
}

func (c *WorkerManager) GetPoolInfo() (poolID, poolName string) {
	if c.Worker.Pool != nil {
		return c.Worker.Pool.PoolId.String(), c.Worker.Pool.PoolTag
	}
	return "", ""
}
