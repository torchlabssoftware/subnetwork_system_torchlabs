package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/google/uuid"

	"github.com/torchlabssoftware/subnetwork_system/internal/db/repository"
	functions "github.com/torchlabssoftware/subnetwork_system/internal/server/functions"
	middleware "github.com/torchlabssoftware/subnetwork_system/internal/server/middleware"
	server "github.com/torchlabssoftware/subnetwork_system/internal/server/models"
	"github.com/torchlabssoftware/subnetwork_system/internal/server/service"
	wsm "github.com/torchlabssoftware/subnetwork_system/internal/server/websocket"
)

type WorkerHandler struct {
	queries   *repository.Queries
	db        *sql.DB
	wsManager *wsm.WebsocketManager
	analytics service.AnalyticsService
}

func NewWorkerHandler(q *repository.Queries, db *sql.DB, analytics service.AnalyticsService) *WorkerHandler {
	w := &WorkerHandler{
		queries:   q,
		db:        db,
		analytics: analytics,
	}
	w.wsManager = wsm.NewWebsocketManager(q, analytics)
	return w
}

func (wh *WorkerHandler) AdminRoutes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.AdminAuthentication)

	r.Post("/", wh.AddWorker)
	r.Get("/", wh.GetAllWorkers)
	r.Get("/{name}", wh.GetWorkerByName)
	r.Delete("/{name}", wh.DeleteWorker)
	r.Post("/{name}/domains", wh.AddWorkerDomain)
	r.Delete("/{name}/domains", wh.DeleteWorkerDomain)
	return r
}

func (wh *WorkerHandler) WorkerRoutes() http.Handler {
	r := chi.NewRouter()

	r.Post("/ws/login", middleware.WorkerAuthentication(wh.login))
	r.Get("/ws/serve", wh.serveWS)
	return r
}

func (wh *WorkerHandler) login(w http.ResponseWriter, r *http.Request) {
	var req server.WorkerLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	if req.WorkerId == nil || *req.WorkerId == uuid.Nil {
		functions.RespondwithError(w, http.StatusBadRequest, "WorkerId is required", fmt.Errorf("worker_id is required"))
		return
	}
	_, err := wh.queries.GetWorkerById(r.Context(), *req.WorkerId)
	if err != nil {
		if err == sql.ErrNoRows {
			functions.RespondwithError(w, http.StatusNotFound, "Worker not found", err)
			return
		}
		functions.RespondwithError(w, http.StatusInternalServerError, "Failed to get worker", err)
		return
	}
	otp := wh.wsManager.OtpMap.NewOTP()
	resp := server.WorkerLoginResponce{
		Otp: otp.Key,
	}
	functions.RespondwithJSON(w, http.StatusOK, resp)
}

func (wh *WorkerHandler) serveWS(w http.ResponseWriter, r *http.Request) {
	otp := r.URL.Query().Get("otp")
	if otp == "" {
		functions.RespondwithError(w, http.StatusBadRequest, "OTP is required", fmt.Errorf("otp is required"))
		return
	}
	if !wh.wsManager.OtpMap.VerifyOTP(otp) {
		functions.RespondwithError(w, http.StatusUnauthorized, "Invalid OTP", fmt.Errorf("invalid otp"))
		return
	}
	wh.wsManager.ServeWS(w, r)
}

func (wh *WorkerHandler) AddWorker(w http.ResponseWriter, r *http.Request) {
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
	if req.Port == nil || *req.Port == 0 {
		functions.RespondwithError(w, http.StatusBadRequest, "Port is required", fmt.Errorf("port is required"))
		return
	}
	if req.PoolId == nil || *req.PoolId == uuid.Nil {
		functions.RespondwithError(w, http.StatusBadRequest, "PoolId is required", fmt.Errorf("pool_id is required"))
		return
	}

	worker, err := wh.queries.CreateWorker(r.Context(), repository.CreateWorkerParams{
		Name:      *req.Name,
		Name_2:    *req.RegionName,
		IpAddress: *req.IPAddress,
		Port:      *req.Port,
		PoolID:    *req.PoolId,
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
		Port:       worker.Port,
		PoolId:     worker.PoolID,
		LastSeen:   worker.LastSeen.Format("2006-01-02T15:04:05.999999Z"),
		CreatedAt:  worker.CreatedAt.Format("2006-01-02T15:04:05.999999Z"),
		Domains:    []string{},
	}

	functions.RespondwithJSON(w, http.StatusCreated, resp)

}

func (wh *WorkerHandler) GetAllWorkers(w http.ResponseWriter, r *http.Request) {
	workers, err := wh.queries.GetAllWorkers(r.Context())
	if err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "Failed to get workers", err)
		return
	}

	resp := []server.AddWorkerResponse{}
	for _, worker := range workers {
		resp = append(resp, server.AddWorkerResponse{
			ID:         worker.ID.String(),
			Name:       worker.Name,
			RegionName: worker.RegionName,
			IpAddress:  worker.IpAddress,
			Status:     worker.Status,
			Port:       worker.Port,
			PoolId:     worker.PoolID,
			LastSeen:   worker.LastSeen.Format("2006-01-02T15:04:05.999999Z"),
			CreatedAt:  worker.CreatedAt.Format("2006-01-02T15:04:05.999999Z"),
			Domains:    worker.Domains,
		})
	}

	functions.RespondwithJSON(w, http.StatusOK, resp)
}

func (wh *WorkerHandler) GetWorkerByName(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		functions.RespondwithError(w, http.StatusBadRequest, "Worker name is required", fmt.Errorf("name is required"))
		return
	}

	worker, err := wh.queries.GetWorkerByName(r.Context(), name)
	if err != nil {
		if err == sql.ErrNoRows {
			functions.RespondwithError(w, http.StatusNotFound, "Worker not found", err)
			return
		}
		functions.RespondwithError(w, http.StatusInternalServerError, "Failed to get worker", err)
		return
	}

	resp := server.AddWorkerResponse{
		ID:         worker.ID.String(),
		Name:       worker.Name,
		RegionName: worker.RegionName,
		IpAddress:  worker.IpAddress,
		Status:     worker.Status,
		Port:       worker.Port,
		PoolId:     worker.PoolID,
		LastSeen:   worker.LastSeen.Format("2006-01-02T15:04:05.999999Z"),
		CreatedAt:  worker.CreatedAt.Format("2006-01-02T15:04:05.999999Z"),
		Domains:    worker.Domains,
	}

	functions.RespondwithJSON(w, http.StatusOK, resp)
}

func (wh *WorkerHandler) DeleteWorker(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		functions.RespondwithError(w, http.StatusBadRequest, "Worker name is required", fmt.Errorf("name is required"))
		return
	}

	err := wh.queries.DeleteWorkerByName(r.Context(), name)
	if err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "Failed to delete worker", err)
		return
	}

	functions.RespondwithJSON(w, http.StatusOK, map[string]string{"message": "Worker deleted successfully"})
}

func (wh *WorkerHandler) AddWorkerDomain(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		functions.RespondwithError(w, http.StatusBadRequest, "Worker name is required", fmt.Errorf("name is required"))
		return
	}

	var req server.AddWorkerDomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if len(req.Domain) == 0 {
		functions.RespondwithError(w, http.StatusBadRequest, "Domains are required", fmt.Errorf("domains are required"))
		return
	}

	_, err := wh.queries.AddWorkerDomain(r.Context(), repository.AddWorkerDomainParams{
		Name:    name,
		Column2: req.Domain,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			functions.RespondwithError(w, http.StatusNotFound, "Worker not found", err)
			return
		}
		functions.RespondwithError(w, http.StatusInternalServerError, "Failed to add domain", err)
		return
	}

	functions.RespondwithJSON(w, http.StatusCreated, map[string]string{"message": "Domains added successfully"})
}

func (wh *WorkerHandler) DeleteWorkerDomain(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		functions.RespondwithError(w, http.StatusBadRequest, "Worker name is required", fmt.Errorf("name is required"))
		return
	}

	var req server.DeleteWorkerDomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if len(req.Domain) == 0 {
		functions.RespondwithError(w, http.StatusBadRequest, "Domains are required", fmt.Errorf("domains are required"))
		return
	}

	err := wh.queries.DeleteWorkerDomain(r.Context(), repository.DeleteWorkerDomainParams{
		Name:    name,
		Column2: req.Domain,
	})
	if err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "Failed to delete domain", err)
		return
	}

	functions.RespondwithJSON(w, http.StatusOK, map[string]string{"message": "Domain deleted successfully"})
}
