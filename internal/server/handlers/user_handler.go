package server

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/torchlabssoftware/subnetwork_system/internal/db/repository"
	server "github.com/torchlabssoftware/subnetwork_system/internal/server/models"
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

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req server.CreateUserRequest

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

	var dataLimit int64
	if req.DataLimit != nil {
		dataLimit = *req.DataLimit
	} else {
		dataLimit = 0
	}

	createUserParams := repository.CreateUserParams{
		Email:     email,
		Username:  uuid.New().String()[:8],
		Password:  uuid.New().String()[:8],
		DataLimit: dataLimit,
	}

	user, err := h.queries.CreateUser(r.Context(), createUserParams)
	if err != nil {
		http.Error(w, "failed to create user", http.StatusInternalServerError)
		log.Printf("cant create new user:%v", err)
		return
	}

	var allowPools []string
	if req.AllowPools != nil && len(*req.AllowPools) > 0 {
		allowPools = *req.AllowPools
	} else {
		allowPools = nil
	}

	pools, err := h.queries.GetPoolsbyTags(r.Context(), allowPools)
	if err != nil {
		http.Error(w, "failed to create user", http.StatusInternalServerError)
		log.Printf("cant create new user:%v", err)
		return
	}

	for _, v := range pools {

		insertUserPoolParams := repository.InsertUserPoolParams{
			PoolID: v.ID,
			UserID: user.ID,
		}

		_, err = h.queries.InsertUserPool(r.Context(), insertUserPoolParams)
		if err != nil {
			http.Error(w, "failed to create user", http.StatusInternalServerError)
			log.Printf("cant create new user:%v", err)
			return
		}
	}

	var ipWhitelist []string
	if req.IpWhiteList != nil && len(*req.IpWhiteList) > 0 {
		ipWhitelist = *req.IpWhiteList
	} else {
		ipWhitelist = nil
	}

	for _, v := range ipWhitelist {
		userIpWhitelistParams := repository.InsertUserIpwhitelistParams{
			UserID: user.ID,
			IpCidr: v,
		}

		_, err = h.queries.InsertUserIpwhitelist(r.Context(), userIpWhitelistParams)
		if err != nil {
			http.Error(w, "failed to create user", http.StatusInternalServerError)
			log.Printf("cant create new user:%v", err)
			return
		}
	}

	responce := server.CreateUserResponce{
		Username:    &user.Username,
		Password:    &user.Password,
		DataLimit:   &user.DataLimit,
		IpWhitelist: &ipWhitelist,
		AllowPools:  &allowPools,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responce)
}
