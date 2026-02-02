package manager

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func TestWebsocketManager_NewWebsocketManager(t *testing.T) {
	worker := &WorkerManager{}
	conn := &websocket.Conn{}
	wm := NewWebsocketManager(worker, conn)
	if wm == nil {
		t.Error("WebsocketManager should not be nil")
	}
	if wm.worker != worker {
		t.Error("Worker should be set")
	}
	if wm.websocketClient == nil {
		t.Error("WebSocket client should be initialized")
	}
}

func TestWebsocketManager_ProcessEvent_Config_Failure(t *testing.T) {
	worker := &WorkerManager{}
	conn := &websocket.Conn{}
	wm := NewWebsocketManager(worker, conn)
	response := Response{
		Success: false,
		Payload: nil,
	}
	wm.processConfig(response)
}

func TestWebsocketManager_ProcessEvent_Config_InvalidJSON(t *testing.T) {
	worker := &WorkerManager{}
	conn := &websocket.Conn{}
	wm := NewWebsocketManager(worker, conn)
	response := Response{
		Success: true,
		Payload: make(chan int),
	}
	wm.processConfig(response)
}

func TestWebsocketManager_ProcessEvent_VerifyUser_Failure(t *testing.T) {
	worker := &WorkerManager{}
	conn := &websocket.Conn{}
	wm := NewWebsocketManager(worker, conn)
	response := Response{
		Success: false,
		Payload: nil,
	}
	wm.processVerifyUserResponse(response)
}

func TestWebsocketManager_ProcessEvent_VerifyUser_InvalidJSON(t *testing.T) {
	worker := &WorkerManager{}
	conn := &websocket.Conn{}
	wm := NewWebsocketManager(worker, conn)
	response := Response{
		Success: true,
		Payload: make(chan int),
	}
	wm.processVerifyUserResponse(response)
}

func TestWebsocketManager_ProcessEvent_UserChange_Failure(t *testing.T) {
	worker := &WorkerManager{}
	conn := &websocket.Conn{}
	wm := NewWebsocketManager(worker, conn)
	response := Response{
		Success: false,
		Payload: nil,
	}
	wm.processUserChange(response)
}

func TestWebsocketManager_ProcessEvent_UserChange_InvalidJSON(t *testing.T) {
	worker := &WorkerManager{}
	conn := &websocket.Conn{}
	wm := NewWebsocketManager(worker, conn)
	response := Response{
		Success: true,
		Payload: make(chan int),
	}
	wm.processUserChange(response)
}

func TestWebsocketManager_ProcessEvent_PoolChange_Failure(t *testing.T) {
	worker := &WorkerManager{}
	conn := &websocket.Conn{}
	wm := NewWebsocketManager(worker, conn)
	response := Response{
		Success: false,
		Payload: nil,
	}
	wm.processPoolChange(response)
}

func TestWebsocketManager_ProcessEvent_PoolChange_InvalidJSON(t *testing.T) {
	worker := &WorkerManager{}
	conn := &websocket.Conn{}
	wm := NewWebsocketManager(worker, conn)
	response := Response{
		Success: true,
		Payload: make(chan int),
	}
	wm.processPoolChange(response)
}

func TestWebsocketManager_HandleEvent_Config(t *testing.T) {
	worker := &WorkerManager{}
	conn := &websocket.Conn{}
	wm := NewWebsocketManager(worker, conn)
	event := Event{
		Type:    "config",
		Payload: createTestConfigPayload(),
	}
	wm.HandleEvent(event)
}

func TestWebsocketManager_HandleEvent_VerifyUser(t *testing.T) {
	worker := &WorkerManager{}
	conn := &websocket.Conn{}
	wm := NewWebsocketManager(worker, conn)
	event := Event{
		Type: "login_success",
		Payload: UserPayload{
			ID:          uuid.New(),
			Username:    "testuser",
			Password:    "testpass",
			Status:      "active",
			IpWhitelist: []string{"127.0.0.1"},
			Pools:       []string{"test-pool:1000000:0"},
		},
	}
	wm.HandleEvent(event)
}

func TestWebsocketManager_HandleEvent_UserChange(t *testing.T) {
	worker := &WorkerManager{}
	conn := &websocket.Conn{}
	wm := NewWebsocketManager(worker, conn)
	event := Event{
		Type:    "user_change",
		Payload: "testuser",
	}
	wm.HandleEvent(event)
}

func TestWebsocketManager_HandleEvent_PoolChange(t *testing.T) {
	worker := &WorkerManager{}
	conn := &websocket.Conn{}
	wm := NewWebsocketManager(worker, conn)
	event := Event{
		Type:    "pool_change",
		Payload: uuid.New(),
	}
	wm.HandleEvent(event)
}

func TestWebsocketManager_HandleEvent_Error(t *testing.T) {
	worker := &WorkerManager{}
	conn := &websocket.Conn{}
	wm := NewWebsocketManager(worker, conn)
	event := Event{
		Type:    "error",
		Payload: "Test error message",
	}
	wm.HandleEvent(event)
}

func TestWebsocketManager_HandleEvent_Unknown(t *testing.T) {
	worker := &WorkerManager{}
	conn := &websocket.Conn{}
	wm := NewWebsocketManager(worker, conn)
	event := Event{
		Type:    "unknown_event",
		Payload: "some data",
	}
	wm.HandleEvent(event)
}

func TestWebsocketManager_WriteEvent(t *testing.T) {
	worker := &WorkerManager{}
	conn := &websocket.Conn{}
	wm := NewWebsocketManager(worker, conn)
	event := Event{
		Type:    "test_event",
		Payload: "test data",
	}
	wm.WriteEvent(event)
}

func TestWebsocketManager_JSONProcessing(t *testing.T) {
	worker := &WorkerManager{}
	conn := &websocket.Conn{}
	wm := NewWebsocketManager(worker, conn)
	configPayload := createTestConfigPayload()
	data, err := json.Marshal(configPayload)
	if err != nil {
		t.Errorf("Failed to marshal config payload: %v", err)
	}
	var resp Response
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Errorf("Failed to unmarshal config: %v", err)
	}
	wm.processConfig(resp)
}

func TestWebsocketManager_EventProcessing_Integration(t *testing.T) {
	worker := &WorkerManager{}
	conn := &websocket.Conn{}
	wm := NewWebsocketManager(worker, conn)
	events := []Event{
		{Type: "config", Payload: createTestConfigPayload()},
		{Type: "login_success", Payload: UserPayload{
			ID:          uuid.New(),
			Username:    "testuser",
			Password:    "testpass",
			Status:      "active",
			IpWhitelist: []string{"127.0.0.1"},
			Pools:       []string{"test-pool:1000000:0"},
		}},
		{Type: "user_change", Payload: "testuser"},
		{Type: "pool_change", Payload: uuid.New()},
		{Type: "error", Payload: "test error"},
		{Type: "unknown_event", Payload: "test data"},
	}
	for _, event := range events {
		wm.HandleEvent(event)
	}
}

func createTestConfigPayload() ConfigPayload {
	return ConfigPayload{
		WorkerName:    "test-worker",
		Region:        "us-east-1",
		PoolID:        uuid.New(),
		PoolTag:       "test-pool",
		PoolPort:      8080,
		PoolSubdomain: "test",
		Upstreams: []UpstreamConfig{
			{
				UpstreamID:       uuid.New(),
				UpstreamTag:      "test-upstream",
				UpstreamFormat:   "http",
				UpstreamUsername: "user",
				UpstreamPassword: "pass",
				UpstreamHost:     "127.0.0.1",
				UpstreamPort:     3128,
				UpstreamProvider: "test",
				Weight:           1,
			},
		},
	}
}
