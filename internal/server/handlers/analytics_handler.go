package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	functions "github.com/torchlabssoftware/subnetwork_system/internal/server/functions"
	"github.com/torchlabssoftware/subnetwork_system/internal/server/service"
)

type AnalyticsHandler struct {
	service service.AnalyticsService
}

func NewAnalyticsHandler(s service.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{service: s}
}

func (h *AnalyticsHandler) RegisterRoutes(r chi.Router) {
	r.Get("/user/{user_id}/usage", h.GetUserUsage)
}

func (h *AnalyticsHandler) GetUserUsage(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "user_id")
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	granularity := r.URL.Query().Get("granularity")

	if granularity == "" {
		granularity = "hour" // Default to hour for now as only hour is implemented in service example
	}

	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		// Default to last 7 days if invalid
		from = time.Now().AddDate(0, 0, -7)
	}

	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		// Default to now if invalid
		to = time.Now()
	}

	data, err := h.service.GetUserUsage(r.Context(), userID, from, to, granularity)
	if err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "failed to get analytics data", err)
		return
	}

	json.NewEncoder(w).Encode(data)
}
