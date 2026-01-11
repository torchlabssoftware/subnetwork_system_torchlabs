package manager

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Worker struct {
	ID               uuid.UUID
	Name             string
	Region           string
	Pool             *Pool
	CaptainURL       string
	APIKey           string
	WebsocketClient  *WebsocketClient
	websocketManager *WebsocketManager
	UpstreamManager  *UpstreamManager
	HealthCollector  *HealthCollector
	UserManager      *UserManager
	mu               sync.Mutex
	reconnect        bool
	ctx              context.Context
	cancel           context.CancelFunc
}

func NewWorker(workerID, baseURL, apiKey string) (*Worker, error) {
	workerUUID, err := uuid.Parse(workerID)
	if err != nil {
		return nil, fmt.Errorf("invalid worker ID: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	upstreamManager := NewUpstreamManager()
	healthCollector := NewHealthCollector(workerUUID, "", "", upstreamManager)

	w := &Worker{
		ID:              workerUUID,
		CaptainURL:      baseURL,
		APIKey:          apiKey,
		reconnect:       true,
		UpstreamManager: upstreamManager,
		HealthCollector: healthCollector,
		ctx:             ctx,
		cancel:          cancel,
	}

	// Initialize UserManager with callback to send events
	w.UserManager = NewUserManager(func(event Event) {
		w.mu.Lock()
		client := w.WebsocketClient
		w.mu.Unlock()
		if client != nil {
			select {
			case client.egress <- event:
			case <-w.ctx.Done():
			}
		}
	})

	// Initialize WebsocketManager with all dependencies
	w.websocketManager = NewWebsocketManager(w.UserManager, upstreamManager, healthCollector, w)

	return w, nil
}

func (c *Worker) Start() {
	// Start healthcollector for periodic sampling
	c.HealthCollector.Start()

	// Start health telemetry sender (every 1 hour)
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

	// Start connection manager with reconnection logic
	go func() {
		for {
			select {
			case <-c.ctx.Done():
				return
			default:
				if !c.reconnect {
					return
				}
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

// Stop gracefully shuts down the worker
func (c *Worker) Stop() {
	log.Println("[Captain] Shutting down worker...")
	c.reconnect = false
	c.cancel()
	c.HealthCollector.Stop()

	c.mu.Lock()
	if c.WebsocketClient != nil {
		c.WebsocketClient.Close()
	}
	c.mu.Unlock()
	log.Println("[Captain] Worker shutdown complete")
}

func (c *Worker) Connect() error {
	otp, err := c.login()
	if err != nil {
		return fmt.Errorf("login failed: %v", err)
	}

	wsURL, err := url.Parse(c.CaptainURL)
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
	header.Set("Authorization", "ApiKey "+c.APIKey)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL.String(), header)
	if err != nil {
		return fmt.Errorf("websocket dial failed: %v", err)
	}

	// Create new WebsocketClient with current connection
	client := NewWebsocketClient(conn, c.websocketManager.HandleEvent)

	c.mu.Lock()
	c.WebsocketClient = client
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		if c.WebsocketClient == client {
			c.WebsocketClient = nil
		}
		c.mu.Unlock()
		client.Close()
	}()

	log.Println("[Captain] WebSocket connected successfully")

	var wg sync.WaitGroup
	wg.Add(2)
	go client.ReadMessage(&wg)
	go client.WriteMessage(&wg)
	wg.Wait()

	return fmt.Errorf("connection closed")
}

func (c *Worker) login() (string, error) {
	loginURL := fmt.Sprintf("%s/worker/ws/login", c.CaptainURL)
	body, _ := json.Marshal(WorkerLoginRequest{WorkerID: c.ID.String()})

	req, err := http.NewRequest(http.MethodPost, loginURL, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "ApiKey "+c.APIKey)

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

func (c *Worker) processConfig(cfg ConfigPayload) {
	c.Name = cfg.WorkerName
	c.Region = cfg.Region
	c.Pool = NewPool(cfg.PoolID, cfg.PoolTag, cfg.PoolPort, cfg.PoolSubdomain)
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
	c.HealthCollector.UpdateWorkerInfo(cfg.WorkerName, c.Pool.Region)
	log.Printf("[Captain] Configuration received for Pool: %s", cfg.PoolTag)
	log.Printf("[Captain] Upstreams count: %d", len(cfg.Upstreams))
}

func (c *Worker) GetPoolInfo() (poolID, poolName string) {
	if c.Pool != nil {
		return c.Pool.PoolId.String(), c.Pool.PoolTag
	}
	return "", ""
}

// SendDataUsage sends a user data usage event to Captain via WebSocket
func (c *Worker) SendDataUsage(usage UserDataUsage) {
	if c.WebsocketClient == nil {
		log.Printf("[DataUsage] WebSocket not connected, cannot send data usage")
		return
	}

	event := Event{
		Type:    "telemetry_usage",
		Payload: usage,
	}

	c.WebsocketClient.egress <- event
	log.Printf("[DataUsage] Sent usage: user=%s, bytes_sent=%d, bytes_received=%d, dest=%s:%d",
		usage.Username, usage.BytesSent, usage.BytesReceived, usage.DestinationHost, usage.DestinationPort)
}

// SendHealthTelemetry sends worker health telemetry to Captain via WebSocket
func (c *Worker) SendHealthTelemetry() {
	if c.WebsocketClient == nil {
		log.Printf("[HealthTelemetry] WebSocket not connected, cannot send health telemetry")
		return
	}

	health := c.HealthCollector.BuildWorkerHealth()
	event := Event{
		Type:    "telemetry_health",
		Payload: health,
	}

	c.WebsocketClient.egress <- event
	log.Printf("[HealthTelemetry] Sent health: status=%s, cpu=%.2f%%, mem=%.2f%%, active_conns=%d, throughput=%d bytes/sec",
		health.Status, health.CpuUsage, health.MemoryUsage, health.ActiveConnections, health.BytesThroughputPerSec)
}
