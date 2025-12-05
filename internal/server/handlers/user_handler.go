package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/torchlabssoftware/subnetwork_system/internal/db/repository"
	functions "github.com/torchlabssoftware/subnetwork_system/internal/server/functions"
	server "github.com/torchlabssoftware/subnetwork_system/internal/server/models"
)

type UserHandler struct {
	queries *repository.Queries
	db      *sql.DB
}

func NewUserHandler(q *repository.Queries, db *sql.DB) *UserHandler {
	return &UserHandler{
		queries: q,
		db:      db,
	}
}

func (h *UserHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Post("/", h.createUser)
	r.Get("/", h.getUsers)
	r.Get("/{id}", h.getUserbyId)
	r.Patch("/{id}", h.updateUser)
	r.Delete("/{id}", h.deleteUser)
	r.Get("/{id}/data-usage", h.getDataUsage)
	r.Get("/{id}/pools", h.getUserAllowPools)
	//r.Patch("/{id}/pools", h.addUserAllowPool)
	//r.Delete("/{id}/pools", h.removeUserAllowPool)
	return r
}

func (h *UserHandler) createUser(w http.ResponseWriter, r *http.Request) {

	//begin transaction
	ctx, err := h.db.Begin()
	if err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "failed to create user", err)
		return
	}
	defer func() {
		_ = ctx.Rollback()
	}()

	qtx := h.queries.WithTx(ctx)

	//get responce
	var req server.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	//validate email
	var email sql.NullString
	if req.Email != nil && *req.Email != "" {
		mail, err := mail.ParseAddress(*req.Email)
		if err != nil {
			functions.RespondwithError(w, http.StatusBadRequest, "invalid email", err)
			return
		}
		email = sql.NullString{String: mail.Address, Valid: true}
	} else {
		email = sql.NullString{Valid: false}
	}

	//validate datalimit
	dataLimit := int64(0)
	if req.DataLimit != nil && *req.DataLimit >= int64(0) {
		dataLimit = *req.DataLimit
	} else {
		functions.RespondwithError(w, http.StatusBadRequest, "send valid data limit", fmt.Errorf("send valid data limit"))
		return
	}

	//insert user data
	createUserParams := repository.CreateUserParams{
		Email:     email,
		Username:  uuid.New().String()[:8],
		Password:  uuid.New().String()[:8],
		DataLimit: dataLimit,
	}

	user, err := qtx.CreateUser(r.Context(), createUserParams)
	if err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "failed to create user", err)
		return
	}

	//insert allow_pool data
	var allowPools []string
	if req.AllowPools != nil && len(*req.AllowPools) > 0 {
		allowPools = *req.AllowPools
	}

	pools, err := qtx.GetPoolsbyTags(r.Context(), allowPools)
	if err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "failed to create user", err)
		return
	}

	insertUserPoolParams := repository.InsertUserPoolParams{
		UserID:  user.ID,
		Column2: pools,
	}

	_, err = qtx.InsertUserPool(r.Context(), insertUserPoolParams)
	if err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "failed to create user", err)
		return
	}

	//insert ipwhilist data
	var ipWhitelist []string
	if req.IpWhiteList != nil && len(*req.IpWhiteList) > 0 {
		ipWhitelist = *req.IpWhiteList
	}

	userIpWhitelistParams := repository.InsertUserIpwhitelistParams{
		UserID:  user.ID,
		Column2: ipWhitelist,
	}

	_, err = qtx.InsertUserIpwhitelist(r.Context(), userIpWhitelistParams)
	if err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "failed to create user", err)
		return
	}

	if err := ctx.Commit(); err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "failed to create user", err)
		return
	}

	responce := server.CreateUserResponce{
		Id:          user.ID,
		Username:    user.Username,
		Password:    user.Password,
		DataLimit:   user.DataLimit,
		IpWhitelist: ipWhitelist,
		AllowPools:  allowPools,
	}

	functions.RespondwithJSON(w, http.StatusCreated, responce)
}

func (h *UserHandler) getUserbyId(w http.ResponseWriter, r *http.Request) {

	//get user params and get user data
	userId := chi.URLParam(r, "id")
	//userId := queryParams.Get("user-id")
	userIdUUID, err := uuid.Parse(userId)
	if err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "incorrect user id", err)
		return
	}

	user, err := h.queries.GetUserbyId(r.Context(), userIdUUID)
	if err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "cant get user by id", err)
		return
	}

	resp := server.GetUserByIdResponce{
		Id:          user.ID,
		Email:       user.Email.String,
		Username:    user.Username,
		Password:    user.Password,
		Data_limit:  user.DataLimit,
		Data_usage:  user.DataUsage,
		Status:      user.Status,
		IpWhitelist: user.IpWhitelist,
		UserPool:    user.Pools,
		Created_at:  user.CreatedAt,
		Updated_at:  user.UpdatedAt,
	}

	functions.RespondwithJSON(w, http.StatusOK, resp)
}

func (h *UserHandler) getUsers(w http.ResponseWriter, r *http.Request) {

	users, err := h.queries.GetAllusers(r.Context())
	if err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "cant get users by id", err)
		return
	}

	resp := []server.GetUserByIdResponce{}

	for _, user := range users {
		resp = append(resp, server.GetUserByIdResponce{
			Id:          user.ID,
			Email:       user.Email.String,
			Username:    user.Username,
			Password:    user.Password,
			Data_limit:  user.DataLimit,
			Data_usage:  user.DataUsage,
			Status:      user.Status,
			IpWhitelist: user.IpWhitelist,
			UserPool:    user.Pools,
			Created_at:  user.CreatedAt,
			Updated_at:  user.UpdatedAt,
		})
	}

	functions.RespondwithJSON(w, http.StatusOK, resp)
}

func (h *UserHandler) updateUser(w http.ResponseWriter, r *http.Request) {
	//get the user id
	userId := chi.URLParam(r, "id")
	id, err := uuid.Parse(userId)
	if err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "invalid user id", err)
		return
	}

	var req server.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "invalid body", err)
		return
	}

	params := repository.UpdateUserParams{
		ID: id,
	}

	if req.Email != nil {
		mail, err := mail.ParseAddress(*req.Email)
		if err != nil {
			functions.RespondwithError(w, http.StatusBadRequest, "enter correct email", err)
			return
		}
		params.Email = sql.NullString{String: mail.Address, Valid: true}
	}

	if req.DataLimit != nil {
		params.DataLimit = sql.NullInt64{Int64: *req.DataLimit, Valid: true}
	}

	if req.Status != nil && *req.Status != "" {
		params.Status = sql.NullString{String: *req.Status, Valid: true}
	}

	user, err := h.queries.UpdateUser(r.Context(), params)
	if err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "server error", err)
		return
	}

	resp := server.UpdateUserResponce{
		Id:        user.ID,
		Email:     user.Email.String,
		DataLimit: user.DataLimit,
		Status:    user.Status,
		UpdatedAt: user.UpdatedAt,
	}

	functions.RespondwithJSON(w, http.StatusOK, resp)
}

func (h *UserHandler) deleteUser(w http.ResponseWriter, r *http.Request) {
	//get the user id
	userId := chi.URLParam(r, "id")
	id, err := uuid.Parse(userId)
	if err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "invalid user id", err)
		return
	}

	if err := h.queries.SoftDeleteUser(r.Context(), id); err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "server error", err)
		return
	}

	res := struct {
		Message string `json:"message"`
	}{
		Message: "user deleted",
	}

	functions.RespondwithJSON(w, http.StatusOK, res)
}

func (h *UserHandler) getDataUsage(w http.ResponseWriter, r *http.Request) {
	//get the user id
	userId := chi.URLParam(r, "id")
	id, err := uuid.Parse(userId)
	if err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "invalid user id", err)
		return
	}

	dataUsage, err := h.queries.GetDatausageById(r.Context(), id)
	if err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "server error", err)
		return
	}

	res := server.GetDatausageReponce{
		DataLimit: dataUsage.DataLimit,
		DataUsage: dataUsage.DataUsage,
	}

	functions.RespondwithJSON(w, http.StatusOK, res)
}

func (h *UserHandler) getUserAllowPools(w http.ResponseWriter, r *http.Request) {
	//get the user id
	userId := chi.URLParam(r, "id")
	id, err := uuid.Parse(userId)
	if err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "invalid user id", err)
		return
	}

	userPool, err := h.queries.GetUserPoolsByUserId(r.Context(), id)
	if err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "server error", err)
		return
	}

	resp := server.GetUserPoolResponce{
		UserId: userPool.ID.UUID,
		Pools:  userPool.Column2,
	}

	functions.RespondwithJSON(w, http.StatusOK, resp)

}

/*func (h *UserHandler) addUserAllowPool(w http.ResponseWriter, r *http.Request) {
	//get the user id
	userId := chi.URLParam(r, "id")
	id, err := uuid.Parse(userId)
	if err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "invalid user id", err)
		return
	}
}

func (h *UserHandler) removeUserAllowPool(w http.ResponseWriter, r *http.Request) {
	//get the user id
	userId := chi.URLParam(r, "id")
	id, err := uuid.Parse(userId)
	if err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "invalid user id", err)
		return
	}
}
*/
