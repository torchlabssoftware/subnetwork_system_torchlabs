package manager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	util "github.com/snail007/goproxy/utils"
)

type Worker struct {
	ID                 uuid.UUID
	Name               string
	Region             string
	Pool               *Pool
	CaptainURL         string
	APIKey             string
	WebsocketManager   *WebsocketManager
	UpstreamManager    *UpstreamManager
	HealthCollector    *HealthCollector
	pendingValidations sync.Map
	mu                 sync.Mutex
	Users              util.ConcurrentMap
	reconnect          bool
}

type User struct {
	ID          uuid.UUID
	Username    string
	Status      string
	IpWhitelist []string
	Pools       []PoolLimit
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
		Users:           util.NewConcurrentMap(),
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

	c.WebsocketManager = NewWebsocketManager(c, conn)

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

	payload := UserLoginPayload{
		Username: user,
		Password: pass,
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
	var resp Response
	if err := json.Unmarshal(data, &resp); err != nil {
		log.Printf("[Captain] Failed to parse verify_user_response: %v", err)
		return
	}
	if !resp.Success {
		return
	}
	data, _ = json.Marshal(resp.Payload)
	var userPayload UserPayload
	if err := json.Unmarshal(data, &userPayload); err != nil {
		log.Printf("[Captain] Failed to parse UserPayload: %v", err)
		return
	}
	if ch, ok := c.pendingValidations.Load(userPayload.Username); ok {
		ch.(chan bool) <- resp.Success
		pools := make([]PoolLimit, 0)
		for _, pool := range userPayload.Pools {
			tag := strings.Split(pool, ":")
			DataLimit, _ := strconv.Atoi(tag[1])
			DataUsage, _ := strconv.Atoi(tag[2])
			pools = append(pools, PoolLimit{
				Tag:       tag[0],
				DataLimit: DataLimit,
				DataUsage: DataUsage,
			})
		}
		user := &User{
			ID:          userPayload.ID,
			Username:    userPayload.Username,
			Status:      userPayload.Status,
			IpWhitelist: userPayload.IpWhitelist,
			Pools:       pools,
		}
		c.Users.Set(user.Username, user)
		return
	}
}

func (c *Worker) processConfig(payload interface{}) {
	data, _ := json.Marshal(payload)
	var resp Response
	if err := json.Unmarshal(data, &resp); err != nil {
		log.Printf("[Captain] Failed to parse config: %v", err)
		return
	}
	if !resp.Success {
		return
	}
	data, _ = json.Marshal(resp.Payload)
	var cfg ConfigPayload
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("[Captain] Failed to parse ConfigPayload: %v", err)
		return
	}
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
