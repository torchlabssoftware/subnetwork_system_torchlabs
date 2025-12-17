package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/torchlabssoftware/subnetwork_system/internal/db/repository"
	functions "github.com/torchlabssoftware/subnetwork_system/internal/server/functions"
	"github.com/torchlabssoftware/subnetwork_system/internal/server/service"
)

var (
	websocketUpgrader = &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

type WebsocketManager struct {
	Workers WorkerList
	sync.RWMutex
	Handlers  map[string]EventHandler
	queries   *repository.Queries
	OtpMap    RetentionMap
	analytics service.AnalyticsService
}

func NewWebsocketManager(queries *repository.Queries, analytics service.AnalyticsService) *WebsocketManager {
	w := &WebsocketManager{
		Workers:   make(WorkerList),
		Handlers:  make(map[string]EventHandler),
		queries:   queries,
		OtpMap:    NewRetentionMap(context.Background(), 10*time.Second),
		analytics: analytics,
	}
	w.setupEventHandlers()
	return w
}

func (ws *WebsocketManager) setupEventHandlers() {
	ws.Handlers["login"] = ws.handleLogin
	ws.Handlers["telemetry_usage"] = ws.handleTelemetryUsage
	ws.Handlers["telemetry_health"] = ws.handleTelemetryHealth
}

func (ws *WebsocketManager) RouteEvent(event Event, w *Worker) error {
	if handler, ok := ws.Handlers[event.Type]; ok {
		if err := handler(event, w); err != nil {
			log.Println("Error routing event:", err)
			return err
		}
		return nil
	} else {
		return fmt.Errorf("no such event type")
	}
}

func (ws *WebsocketManager) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "Could not open websocket connection", err)
		return
	}

	log.Println("Worker connected via WebSocket.")

	worker := NewWorker(conn, ws)
	ws.AddWorker(worker)
	go worker.ReadMessage()
	go worker.WriteMessage()
}

func (ws *WebsocketManager) handleLogin(event Event, w *Worker) error {
	var payload loginPayload
	if m, ok := event.Payload.(map[string]interface{}); ok {
		data, err := json.Marshal(m)
		if err != nil {
			return fmt.Errorf("could not marshal payload map: %v", err)
		}

		if err := json.Unmarshal(data, &payload); err != nil {
			return fmt.Errorf("invalid login payload: %v", err)
		}
	} else {
		return fmt.Errorf("invalid payload type: expected map[string]interface{}")
	}

	user, err := ws.queries.GetUserByUsername(context.Background(), payload.Username)
	if err != nil {
		return fmt.Errorf("user login failed: %v", err)
	}

	if user.Password != payload.Password {
		return fmt.Errorf("login failed: incorrect password")
	}

	w.egress <- Event{
		Type:    "login_success",
		Payload: successPayload{Success: true, Payload: user},
	}

	return nil
}

func (ws *WebsocketManager) handleTelemetryUsage(event Event, w *Worker) error {
	var payload service.UserDataUsage
	if m, ok := event.Payload.(map[string]interface{}); ok {
		data, err := json.Marshal(m)
		if err != nil {
			return fmt.Errorf("could not marshal payload map: %v", err)
		}
		if err := json.Unmarshal(data, &payload); err != nil {
			return fmt.Errorf("invalid telemetry usage payload: %v", err)
		}
	} else {
		return fmt.Errorf("invalid payload type: expected map[string]interface{}")
	}

	return ws.analytics.RecordUserDataUsage(context.Background(), payload)
}

func (ws *WebsocketManager) handleTelemetryHealth(event Event, w *Worker) error {
	var payload service.WorkerHealth
	if m, ok := event.Payload.(map[string]interface{}); ok {
		data, err := json.Marshal(m)
		if err != nil {
			return fmt.Errorf("could not marshal payload map: %v", err)
		}
		if err := json.Unmarshal(data, &payload); err != nil {
			return fmt.Errorf("invalid telemetry health payload: %v", err)
		}
	} else {
		return fmt.Errorf("invalid payload type: expected map[string]interface{}")
	}

	return ws.analytics.RecordWorkerHealth(context.Background(), payload)
}

func (ws *WebsocketManager) AddWorker(w *Worker) {
	ws.Lock()
	defer ws.Unlock()
	ws.Workers[w] = true
}

func (ws *WebsocketManager) RemoveWorker(w *Worker) {
	ws.Lock()
	defer ws.Unlock()
	if _, ok := ws.Workers[w]; ok {
		w.Connection.Close()
		delete(ws.Workers, w)
	}
}
