package manager

import (
	"encoding/json"
	"log"
)

type WebsocketManager struct {
	worker          *Worker
	userManager     *UserManager
	upstreamManager *UpstreamManager
	healthCollector *HealthCollector
}

func NewWebsocketManager(userManager *UserManager, upstreamManager *UpstreamManager, healthCollector *HealthCollector, worker *Worker) *WebsocketManager {
	return &WebsocketManager{
		userManager:     userManager,
		upstreamManager: upstreamManager,
		healthCollector: healthCollector,
		worker:          worker,
	}
}

func (m *WebsocketManager) HandleEvent(event Event) {
	log.Printf("[Captain] Received event: %s", event.Type)

	switch event.Type {
	case "config":
		m.processConfig(event.Payload)
	case "login_success":
		m.processVerifyUserResponse(event.Payload)
	case "error":
		log.Printf("[Captain] Error from server: %v", event.Payload)
	default:
		log.Printf("[Captain] Unhandled event type: %s", event.Type)
	}
}

func (m *WebsocketManager) processConfig(payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[Captain] Failed to marshal config payload: %v", err)
		return
	}
	var resp Response
	if err := json.Unmarshal(data, &resp); err != nil {
		log.Printf("[Captain] Failed to parse config: %v", err)
		return
	}
	if !resp.Success {
		log.Printf("[Captain] Config response indicates failure")
		return
	}
	data, err = json.Marshal(resp.Payload)
	if err != nil {
		log.Printf("[Captain] Failed to marshal config payload data: %v", err)
		return
	}
	var cfg ConfigPayload
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("[Captain] Failed to parse ConfigPayload: %v", err)
		return
	}

	m.worker.processConfig(cfg)
}

func (m *WebsocketManager) processVerifyUserResponse(payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[Captain] Failed to marshal verify_user_response payload: %v", err)
		return
	}
	var resp Response
	if err := json.Unmarshal(data, &resp); err != nil {
		log.Printf("[Captain] Failed to parse verify_user_response: %v", err)
		return
	}
	if !resp.Success {
		log.Printf("[Captain] User verification failed")
		return
	}
	data, err = json.Marshal(resp.Payload)
	if err != nil {
		log.Printf("[Captain] Failed to marshal user payload data: %v", err)
		return
	}
	var userPayload UserPayload
	if err := json.Unmarshal(data, &userPayload); err != nil {
		log.Printf("[Captain] Failed to parse UserPayload: %v", err)
		return
	}
	m.userManager.processVerifyUserResponse(userPayload)
}
