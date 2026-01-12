package manager

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
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
	UpstreamManager  *UpstreamManager
	HealthCollector  *HealthCollector
	userManager      *UserManager
	ctx              context.Context
}

func NewWorkerManager(workerID, baseURL, apiKey string) (*WorkerManager, error) {
	workerUUID, err := uuid.Parse(workerID)
	if err != nil {
		return nil, fmt.Errorf("invalid worker ID: %w", err)
	}
	ctx := context.Background()
	upstreamManager := NewUpstreamManager()
	healthCollector := NewHealthCollector(workerUUID, "", "", upstreamManager)
	userManager := NewUserManager()
	w := &WorkerManager{
		Worker: worker{
			ID:         workerUUID,
			CaptainURL: baseURL,
			APIKey:     apiKey,
		},
		UpstreamManager: upstreamManager,
		HealthCollector: healthCollector,
		userManager:     userManager,
		ctx:             ctx,
	}

	return w, nil
}

func (c *WorkerManager) Start() {
	c.HealthCollector.Start()
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.SendHealthTelemetry()
			case <-c.ctx.Done():
				return
			}
		}
	}()
	go func() {
		for {
			select {
			case <-c.ctx.Done():
				return
			default:
			}
			if err := c.Connect(); err != nil {
				log.Printf("[Captain] Connection failed: %v. Retrying in 5s...", err)
				select {
				case <-time.After(5 * time.Second):
				case <-c.ctx.Done():
					return
				}
				continue
			}

			log.Println("[Captain] Disconnected. Reconnecting in 2s...")
			select {
			case <-time.After(2 * time.Second):
			case <-c.ctx.Done():
				return
			}
		}
	}()
}

func (c *WorkerManager) Connect() error {
	otp, err := c.login()
	if err != nil {
		return fmt.Errorf("login failed: %v", err)
	}
	wsURL, err := url.Parse(c.Worker.CaptainURL)
	if err != nil {
		return fmt.Errorf("invalid base URL: %v", err)
	}
	if wsURL.Scheme == "https" {
		wsURL.Scheme = "wss"
	} else {
		wsURL.Scheme = "ws"
	}
	wsURL.Path = "/worker/ws/serve"
	q := wsURL.Query()
	q.Set("otp", otp)
	wsURL.RawQuery = q.Encode()
	log.Printf("[Captain] Connecting to WebSocket: %s", wsURL.String())
	header := http.Header{}
	header.Set("Authorization", "ApiKey "+c.Worker.APIKey)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL.String(), header)
	if err != nil {
		return fmt.Errorf("websocket dial failed: %v", err)
	}
	c.websocketManager = NewWebsocketManager(c, conn)
	defer func() {
		c.websocketManager.Stop()
	}()
	c.websocketManager.Start()
	log.Println("[Captain] WebSocket connected successfully")
	return fmt.Errorf("connection closed")
}

func (c *WorkerManager) login() (string, error) {
	loginURL := fmt.Sprintf("%s/worker/ws/login", c.Worker.CaptainURL)
	body, _ := json.Marshal(WorkerLoginRequest{WorkerID: c.Worker.ID.String()})
	req, err := http.NewRequest(http.MethodPost, loginURL, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "ApiKey "+c.Worker.APIKey)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("server returned status: %d", resp.StatusCode)
	}
	var loginResp WorkerLoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return "", err
	}
	return loginResp.Otp, nil
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
	c.UpstreamManager.SetUpstreams(upstreams)
	c.HealthCollector.UpdateWorkerInfo(cfg.WorkerName, c.Worker.Pool.Region)
	log.Printf("[Captain] Configuration received for Pool: %s", cfg.PoolTag)
	log.Printf("[Captain] Upstreams count: %d", len(cfg.Upstreams))
}

func (c *WorkerManager) VerifyUser(user, pass string) bool {
	return c.userManager.VerifyUser(user, pass)
}

func (c *WorkerManager) processVerifyUserResponse(userPayload UserPayload) {
	c.userManager.processVerifyUserResponse(userPayload)
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
