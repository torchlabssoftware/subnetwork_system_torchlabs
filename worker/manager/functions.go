package manager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
)

func LogintoCaptain(captainURL, workerID, APIKey string) (string, error) {
	loginURL := fmt.Sprintf("%s/worker/ws/login", captainURL)
	body, _ := json.Marshal(WorkerLoginRequest{WorkerID: workerID})
	req, err := http.NewRequest(http.MethodPost, loginURL, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "ApiKey "+APIKey)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", err
	}
	var loginResp WorkerLoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return "", err
	}
	return loginResp.Otp, nil
}

func ConnnectToWebsocket(captainURL, APIKey, otp string) (*websocket.Conn, error) {
	wsURL, err := url.Parse(captainURL)
	if err != nil {
		return nil, err
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
	header.Set("Authorization", "ApiKey "+APIKey)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL.String(), header)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
