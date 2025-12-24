package websocket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

func NewCaptainClient(baseURL, workerID, apiKey string) *CaptainClient {
	return &CaptainClient{
		BaseURL:   baseURL,
		WorkerID:  workerID,
		APIKey:    apiKey,
		reconnect: true,
	}
}

func (c *CaptainClient) Start() {
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

func (c *CaptainClient) Connect() error {
	otp, err := c.login()
	if err != nil {
		return fmt.Errorf("login failed: %v", err)
	}

	wsURL, err := url.Parse(c.BaseURL)
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
	c.Conn = conn
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		c.Conn.Close()
		c.Conn = nil
		c.mu.Unlock()
	}()

	log.Println("[Captain] WebSocket connected successfully")

	for {
		var event Event
		if err := conn.ReadJSON(&event); err != nil {
			return fmt.Errorf("read error: %v", err)
		}
		c.handleEvent(event)
	}
}

func (c *CaptainClient) login() (string, error) {
	loginURL := fmt.Sprintf("%s/worker/ws/login", c.BaseURL)
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

func (c *CaptainClient) Send(event Event) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Conn == nil {
		return fmt.Errorf("not connected")
	}
	return c.Conn.WriteJSON(event)
}

func (c *CaptainClient) handleEvent(event Event) {
	log.Printf("[Captain] Received event: %s", event.Type)

	switch event.Type {
	case "config":
		c.processConfig(event.Payload)
	case "login_success":
		log.Println("[Captain] Login confirmed by server")
	case "error":
		log.Printf("[Captain] Error from server: %v", event.Payload)
	default:
		log.Printf("[Captain] Unhandled event type: %s", event.Type)
	}
}

func (c *CaptainClient) processConfig(payload interface{}) {
	data, _ := json.Marshal(payload)
	var config ConfigPayload
	if err := json.Unmarshal(data, &config); err != nil {
		log.Printf("[Captain] Failed to parse config: %v", err)
		return
	}
	log.Printf("[Captain] Configuration received for Pool: %s (Port: %d)", config.PoolTag, config.PoolPort)
	log.Printf("[Captain] Upstreams count: %d", len(config.Upstreams))
}
