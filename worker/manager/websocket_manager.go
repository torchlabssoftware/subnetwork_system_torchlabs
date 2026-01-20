package manager

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type WebsocketManager struct {
	worker          *WorkerManager
	websocketClient *WebsocketClient
}

func NewWebsocketManager(worker *WorkerManager, conn *websocket.Conn) *WebsocketManager {
	websocketManager := &WebsocketManager{
		worker: worker,
	}
	websocketManager.websocketClient = NewWebsocketClient(conn, websocketManager.HandleEvent)
	return websocketManager
}

func (m *WebsocketManager) Start() {
	var wg sync.WaitGroup
	wg.Add(2)
	go m.websocketClient.ReadMessage(&wg)
	go m.websocketClient.WriteMessage(&wg)
	log.Println("[worker] WebSocket connected successfully")
	wg.Wait()
}

func (m *WebsocketManager) Stop() {
	m.websocketClient.Close()
}

func (w *WebsocketManager) WriteEvent(event Event) {
	log.Println("[websocket] sending event: ", event.Type)
	w.websocketClient.egress <- event
}

func (m *WebsocketManager) HandleEvent(event Event) {
	log.Printf("[websocket] Received event: %s", event.Type)
	switch event.Type {
	case "config":
		m.processConfig(event.Payload)
	case "login_success":
		m.processVerifyUserResponse(event.Payload)
	case "user_change":
		m.processUserChange(event.Payload)
	case "pool_change":
		m.processPoolChange(event.Payload)
	case "error":
		log.Printf("[websocket] Error from server: %v", event.Payload)
	default:
		log.Printf("[websocket] Unhandled event type: %s", event.Type)
	}
}

func (m *WebsocketManager) processConfig(payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[websocket] Failed to marshal config payload: %v", err)
		return
	}
	var resp Response
	if err := json.Unmarshal(data, &resp); err != nil {
		log.Printf("[websocket] Failed to parse config: %v", err)
		return
	}
	if !resp.Success {
		log.Printf("[websocket] Config response indicates failure")
		return
	}
	data, err = json.Marshal(resp.Payload)
	if err != nil {
		log.Printf("[websocket] Failed to marshal config payload data: %v", err)
		return
	}
	var cfg ConfigPayload
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("[websocket] Failed to parse ConfigPayload: %v", err)
		return
	}

	m.worker.processConfig(cfg)
}

func (m *WebsocketManager) processVerifyUserResponse(payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[websocket] Failed to marshal verify_user_response payload: %v", err)
		return
	}
	var resp Response
	if err := json.Unmarshal(data, &resp); err != nil {
		log.Printf("[websocket] Failed to parse verify_user_response: %v", err)
		return
	}
	if !resp.Success {
		log.Printf("[websocket] User verification failed")
		return
	}
	data, err = json.Marshal(resp.Payload)
	if err != nil {
		log.Printf("[websocket] Failed to marshal user payload data: %v", err)
		return
	}
	var userPayload UserPayload
	if err := json.Unmarshal(data, &userPayload); err != nil {
		log.Printf("[websocket] Failed to parse UserPayload: %v", err)
		return
	}
	m.worker.processVerifyUserResponse(userPayload)
}

func (m *WebsocketManager) processUserChange(payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[websocket] Failed to marshal user_change payload: %v", err)
		return
	}
	var resp Response
	if err := json.Unmarshal(data, &resp); err != nil {
		log.Printf("[websocket] Failed to parse user_change: %v", err)
		return
	}
	if !resp.Success {
		log.Printf("[websocket] User change response indicates failure")
		return
	}
	data, err = json.Marshal(resp.Payload)
	if err != nil {
		log.Printf("[websocket] Failed to marshal user_change payload data: %v", err)
		return
	}
	var username string
	if err := json.Unmarshal(data, &username); err != nil {
		log.Printf("[websocket] Failed to parse UserPayload: %v", err)
		return
	}
	m.worker.processUserChange(username)
}

func (m *WebsocketManager) processPoolChange(payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[websocket] Failed to marshal pool_change payload: %v", err)
		return
	}
	var resp Response
	if err := json.Unmarshal(data, &resp); err != nil {
		log.Printf("[websocket] Failed to parse pool_change: %v", err)
		return
	}
	if !resp.Success {
		log.Printf("[websocket] Pool change response indicates failure")
		return
	}
	data, err = json.Marshal(resp.Payload)
	if err != nil {
		log.Printf("[websocket] Failed to marshal pool_change payload data: %v", err)
		return
	}
	var poolId uuid.UUID
	if err := json.Unmarshal(data, &poolId); err != nil {
		log.Printf("[websocket] Failed to parse PoolPayload: %v", err)
		return
	}
	m.worker.processPoolChange(poolId)
}
