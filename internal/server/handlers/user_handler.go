package server

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/torchlabssoftware/subnetwork_system/internal/db/repository"
)

type UserHandler struct {
	queries *repository.Queries
}

func NewUserHandler(q *repository.Queries) *UserHandler {
	return &UserHandler{
		queries: q,
	}
}

func (h *UserHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Post("/", h.CreateUser)
	return r
}

type createUserRequest struct {
	Email     *string `json:"email,omitempty"`
	DataLimit *int64  `json:"data_limit"`
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		log.Printf("cant create new user:%v", err)
		return
	}

	var email sql.NullString
	if req.Email != nil && *req.Email != "" {
		email = sql.NullString{String: *req.Email, Valid: true}
	} else {
		email = sql.NullString{Valid: false}
	}

	if req.DataLimit == nil {
		http.Error(w, "data_limit is required", http.StatusBadRequest)
		return
	}

	params := repository.CreateUserParams{
		ID:        uuid.New(),
		Email:     email,
		Username:  uuid.New().String()[:8],
		Password:  uuid.New().String()[:8],
		DataLimit: *req.DataLimit,
	}

	user, err := h.queries.CreateUser(r.Context(), params)
	if err != nil {
		http.Error(w, "failed to create user", http.StatusInternalServerError)
		log.Printf("cant create new user:%v", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}
