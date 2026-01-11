package manager

import (
	"bytes"
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
}

func NewWorker(workerID, baseURL, apiKey string) *Worker {
	workerUUID, _ := uuid.Parse(workerID)
	upstreamManager := NewUpstreamManager()
	return &Worker{
		ID:              workerUUID,
		CaptainURL:      baseURL,
		APIKey:          apiKey,
		reconnect:       true,
		UpstreamManager: upstreamManager,
		HealthCollector: NewHealthCollector(workerUUID, "", "", upstreamManager),
	}

}

func (c *Worker) Start() {
	//start healthcollector and start function for send telemantry health data for every 1hour
	c.HealthCollector.Start()
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			c.SendHealthTelemetry()
		}
	}()
	//start worker connect
	go func() {
		for c.reconnect {
			if err := c.Connect(); err != nil {
				log.Printf("[Captain] Connection failed: %v. Retrying in 5s...", err)
				time.Sleep(5 * time.Second)
				continue
			}
			log.Println("[Captain] Disconnected. Reconnecting in 2s...")
			time.Sleep(2 * time.Second)
		}
	}()
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

	c.websocketManager = NewWebsocketManager(c.websocketManager.userManager, c.websocketManager.upstreamManager, c.websocketManager.healthCollector, c)
	c.WebsocketClient = NewWebsocketClient(conn, c.websocketManager.HandleEvent)

	c.UserManager = NewUserManager(func(event Event) {
		c.WebsocketClient.egress <- event
	})

	defer func() {
		c.mu.Lock()
		c.WebsocketClient.Connection.Close()
		c.WebsocketClient = nil
		c.mu.Unlock()
	}()

	log.Println("[Captain] WebSocket connected successfully")

	var wg sync.WaitGroup
	wg.Add(2)
	go c.WebsocketClient.ReadMessage(&wg)
	go c.WebsocketClient.WriteMessage(&wg)
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
