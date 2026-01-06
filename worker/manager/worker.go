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
	util "github.com/snail007/goproxy/utils"
)

type Worker struct {
	WorkerID           string
	WorkerName         string
	Region             string
	CaptainURL         string
	APIKey             string
	WebsocketManager   *WebsocketManager
	mu                 sync.Mutex
	reconnect          bool
	pendingValidations sync.Map
	Users              util.ConcurrentMap
	Pool               *Pool
	UpstreamManager    *UpstreamManager
	HealthCollector    *HealthCollector
}

func NewWorker(baseURL, workerID, apiKey string) *Worker {
	workerUUID, _ := uuid.Parse(workerID)
	upstreamMgr := NewUpstreamManager()
	return &Worker{
		CaptainURL:      baseURL,
		WorkerID:        workerID,
		APIKey:          apiKey,
		reconnect:       true,
		Users:           util.NewConcurrentMap(),
		UpstreamManager: upstreamMgr,
		HealthCollector: NewHealthCollector(workerUUID, "", "", upstreamMgr),
	}
}

func (c *Worker) Start() {
	// Start the health collector for periodic sampling
	c.HealthCollector.Start()

	// Start hourly health telemetry reporting
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			c.SendHealthTelemetry()
		}
	}()

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

	c.mu.Lock()
	c.WebsocketManager = NewWebsocketManager(c, conn)
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		c.WebsocketManager.Connection.Close()
		c.WebsocketManager = nil
		c.mu.Unlock()
	}()

	log.Println("[Captain] WebSocket connected successfully")

	var wg sync.WaitGroup
	wg.Add(2)
	go c.WebsocketManager.ReadMessage(&wg)
	go c.WebsocketManager.WriteMessage(&wg)
	wg.Wait()

	return fmt.Errorf("connection closed")
}

func (c *Worker) login() (string, error) {
	loginURL := fmt.Sprintf("%s/worker/ws/login", c.CaptainURL)
	body, _ := json.Marshal(LoginRequest{WorkerID: c.WorkerID})

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

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return "", err
	}

	return loginResp.Otp, nil
}

func (c *Worker) HandleEvent(event Event) {
	log.Printf("[Captain] Received event: %s", event.Type)

	switch event.Type {
	case "config":
		c.processConfig(event.Payload)
	case "login_success":
		c.processVerifyUserResponse(event.Payload)
	case "error":
		log.Printf("[Captain] Error from server: %v", event.Payload)
	default:
		log.Printf("[Captain] Unhandled event type: %s", event.Type)
	}
}

func (c *Worker) VerifyUser(user, pass string) bool {
	respChan := make(chan bool)

	c.pendingValidations.Store(user, respChan)
	defer c.pendingValidations.Delete(user)

	payload := map[string]string{
		"username": user,
		"password": pass,
	}

	c.WebsocketManager.egress <- Event{Type: "verify_user", Payload: payload}

	select {
	case result := <-respChan:
		return result
	case <-time.After(5 * time.Second):
		log.Printf("[Captain] VerifyUser timeout for %s", user)
		return false
	}
}

func (c *Worker) processVerifyUserResponse(payload interface{}) {
	data, _ := json.Marshal(payload)
	var resp struct {
		Success bool        `json:"success"`
		Payload UserPayload `json:"payload"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		log.Printf("[Captain] Failed to parse verify_user_response: %v", err)
		return
	}

	if ch, ok := c.pendingValidations.Load(resp.Payload.Username); ok {
		ch.(chan bool) <- resp.Success
		if resp.Success {
			user := &User{
				ID:          resp.Payload.ID,
				Status:      resp.Payload.Status,
				IpWhitelist: resp.Payload.IpWhitelist,
				Pools:       resp.Payload.Pools,
			}
			c.Users.Set(resp.Payload.Username, user)
			return
		}
	}
}

func (c *Worker) processConfig(payload interface{}) {
	data, _ := json.Marshal(payload)
	var config ConfigPayload
	if err := json.Unmarshal(data, &config); err != nil {
		log.Printf("[Captain] Failed to parse config: %v", err)
		return
	}
	upstreams := make([]Upstream, 0)
	for _, upstream := range config.Upstreams {
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
	c.Pool = NewPool(config.PoolID, config.PoolTag, config.PoolPort, config.PoolSubdomain, upstreams)

	// Update the UpstreamManager with the new upstreams for round-robin load balancing
	c.UpstreamManager.SetUpstreams(upstreams)

	// Update worker name and region in health collector
	c.WorkerName = config.WorkerName
	c.Pool.Region = "" // Region will be set when Captain provides it
	c.HealthCollector.UpdateWorkerInfo(config.WorkerName, c.Pool.Region)

	log.Printf("[Captain] Configuration received for Pool: %s (Port: %d)", config.PoolTag, config.PoolPort)
	log.Printf("[Captain] Upstreams count: %d", len(config.Upstreams))
}

// GetPoolInfo returns the current pool ID and name
func (c *Worker) GetPoolInfo() (poolID, poolName string) {
	if c.Pool != nil {
		return c.Pool.PoolId.String(), c.Pool.PoolTag
	}
	return "", ""
}

// SendDataUsage sends a user data usage event to Captain via WebSocket
func (c *Worker) SendDataUsage(usage UserDataUsage) {
	if c.WebsocketManager == nil {
		log.Printf("[DataUsage] WebSocket not connected, cannot send data usage")
		return
	}

	event := Event{
		Type:    "telemetry_usage",
		Payload: usage,
	}

	c.WebsocketManager.egress <- event
	log.Printf("[DataUsage] Sent usage: user=%s, bytes_sent=%d, bytes_received=%d, dest=%s:%d",
		usage.Username, usage.BytesSent, usage.BytesReceived, usage.DestinationHost, usage.DestinationPort)
}

// SendHealthTelemetry sends worker health telemetry to Captain via WebSocket
func (c *Worker) SendHealthTelemetry() {
	if c.WebsocketManager == nil {
		log.Printf("[HealthTelemetry] WebSocket not connected, cannot send health telemetry")
		return
	}

	health := c.HealthCollector.BuildWorkerHealth()
	event := Event{
		Type:    "telemetry_health",
		Payload: health,
	}

	c.WebsocketManager.egress <- event
	log.Printf("[HealthTelemetry] Sent health: status=%s, cpu=%.2f%%, mem=%.2f%%, active_conns=%d, throughput=%d bytes/sec",
		health.Status, health.CpuUsage, health.MemoryUsage, health.ActiveConnections, health.BytesThroughputPerSec)
}
