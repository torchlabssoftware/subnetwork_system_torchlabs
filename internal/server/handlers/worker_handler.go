package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi"

	"github.com/torchlabssoftware/subnetwork_system/internal/db/repository"
	functions "github.com/torchlabssoftware/subnetwork_system/internal/server/functions"
	middleware "github.com/torchlabssoftware/subnetwork_system/internal/server/middleware"
	server "github.com/torchlabssoftware/subnetwork_system/internal/server/models"
	wsm "github.com/torchlabssoftware/subnetwork_system/internal/server/websocket"
)

type WorkerHandler struct {
	queries   *repository.Queries
	db        *sql.DB
	wsManager *wsm.WebsocketManager
}

func NewWorkerHandler(q *repository.Queries, db *sql.DB) *WorkerHandler {
	w := &WorkerHandler{
		queries: q,
		db:      db,
	}
	w.wsManager = wsm.NewWebsocketManager(q)
	return w
}

func (ws *WorkerHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.AdminAuthentication)

	r.Get("/ws", ws.serveWS)
	r.Post("/", ws.AddWorker)
	return r

}

func (ws *WorkerHandler) serveWS(w http.ResponseWriter, r *http.Request) {
	ws.wsManager.ServeWS(w, r)
}

func (ws *WorkerHandler) AddWorker(w http.ResponseWriter, r *http.Request) {
	var req server.AddWorkerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.Name == nil || *req.Name == "" {
		functions.RespondwithError(w, http.StatusBadRequest, "Name is required", fmt.Errorf("name is required"))
		return
	}
	if req.RegionName == nil || *req.RegionName == "" {
		functions.RespondwithError(w, http.StatusBadRequest, "RegionName is required", fmt.Errorf("region_name is required"))
		return
	}
	if req.IPAddress == nil || *req.IPAddress == "" {
		functions.RespondwithError(w, http.StatusBadRequest, "IPAddress is required", fmt.Errorf("ip_address is required"))
		return
	}

	worker, err := ws.queries.CreateWorker(r.Context(), repository.CreateWorkerParams{
		Name:      *req.Name,
		Name_2:    *req.RegionName,
		IpAddress: *req.IPAddress,
	})
	if err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "Failed to create worker", err)
		return
	}

	resp := server.AddWorkerResponse{
		ID:         worker.ID.String(),
		Name:       worker.Name,
		RegionName: *req.RegionName,
		IpAddress:  worker.IpAddress,
		Status:     worker.Status,
		LastSeen:   worker.LastSeen.Format("2006-01-02T15:04:05.999999Z"),
		CreatedAt:  worker.CreatedAt.Format("2006-01-02T15:04:05.999999Z"),
		UpdatedAt:  worker.UpdatedAt.Format("2006-01-02T15:04:05.999999Z"),
	}

	functions.RespondwithJSON(w, http.StatusCreated, resp)

}
