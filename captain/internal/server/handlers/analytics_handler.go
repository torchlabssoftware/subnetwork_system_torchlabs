package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	functions "github.com/torchlabssoftware/subnetwork_system/internal/server/functions"
	models "github.com/torchlabssoftware/subnetwork_system/internal/server/models"
)

type AnalyticsHandler struct {
	service models.AnalyticsService
}

func NewAnalyticsHandler(s models.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{service: s}
}

func (h *AnalyticsHandler) RegisterRoutes(r chi.Router) {
	r.Get("/user/{user_id}/usage", h.GetUserUsage)
	r.Get("/worker/{worker_id}/health", h.GetWorkerHealth)
	r.Get("/user/{user_id}/website-access", h.GetUserWebsiteAccess)
}

func (h *AnalyticsHandler) GetUserUsage(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "user_id")
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	granularity := r.URL.Query().Get("granularity")
	if granularity == "" {
		granularity = "hour"
	}
	from, err := time.Parse(time.DateOnly, fromStr)
	if err != nil {
		from = time.Now().AddDate(0, 0, -7)
	}
	to, err := time.Parse(time.DateOnly, toStr)
	if err != nil {
		to = time.Now()
	}
	data, err := h.service.GetUserUsage(r.Context(), userID, from, to, granularity)
	if err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "failed to get analytics data", err)
		return
	}
	json.NewEncoder(w).Encode(data)
}

func (h *AnalyticsHandler) GetWorkerHealth(w http.ResponseWriter, r *http.Request) {
	workerID := chi.URLParam(r, "worker_id")
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		from = time.Now().AddDate(0, 0, -7)
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		to = time.Now()
	}
	data, err := h.service.GetWorkerHealth(r.Context(), workerID, from, to)
	if err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "failed to get worker health data", err)
		return
	}
	json.NewEncoder(w).Encode(data)
}

func (h *AnalyticsHandler) GetUserWebsiteAccess(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "user_id")
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		from = time.Now().AddDate(0, 0, -7)
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		to = time.Now()
	}
	data, err := h.service.GetUserWebsiteAccess(r.Context(), userID, from, to)
	if err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "failed to get website access data", err)
		return
	}
	json.NewEncoder(w).Encode(data)
}
