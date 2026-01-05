package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	functions "github.com/torchlabssoftware/subnetwork_system/internal/server/functions"
	middleware "github.com/torchlabssoftware/subnetwork_system/internal/server/middleware"
	models "github.com/torchlabssoftware/subnetwork_system/internal/server/models"
	service "github.com/torchlabssoftware/subnetwork_system/internal/server/service"
)

type UserHandler struct {
	service service.UserService
}

func NewUserHandler(service service.UserService) *UserHandler {
	return &UserHandler{
		service: service,
	}
}

func (h *UserHandler) AdminRoutes() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.AdminAuthentication)

	r.Post("/", h.createUser)
	r.Get("/", h.getUsers)
	r.Get("/{id}", h.getUserbyId)
	r.Patch("/{id}", h.UpdateUserStatus)
	r.Delete("/{id}", h.deleteUser)
	r.Get("/{id}/data-usage", h.getDataUsage)
	r.Get("/{id}/pools", h.getUserAllowPools)
	r.Post("/{id}/pools", h.addUserAllowPool)
	r.Delete("/{id}/pools", h.removeUserAllowPool)
	r.Get("/{id}/ipwhitelist", h.getUserIpWhitelist)
	r.Post("/{id}/ipwhitelist", h.addUserIpWhitelist)
	r.Delete("/{id}/ipwhitelist", h.removeUserIpWhitelist)

	r.Post("/generate", h.GenerateproxyString)

	return r
}

func (h *UserHandler) TestRoutes() http.Handler {
	r := chi.NewRouter()
	r.Post("/", h.createUser)
	r.Get("/", h.getUsers)
	r.Get("/{id}", h.getUserbyId)
	r.Patch("/{id}", h.UpdateUserStatus)
	r.Delete("/{id}", h.deleteUser)
	r.Get("/{id}/data-usage", h.getDataUsage)
	r.Get("/{id}/pools", h.getUserAllowPools)
	r.Post("/{id}/pools", h.addUserAllowPool)
	r.Delete("/{id}/pools", h.removeUserAllowPool)
	r.Get("/{id}/ipwhitelist", h.getUserIpWhitelist)
	r.Post("/{id}/ipwhitelist", h.addUserIpWhitelist)
	r.Delete("/{id}/ipwhitelist", h.removeUserIpWhitelist)
	r.Post("/generate", h.GenerateproxyString)
	return r
}

func (h *UserHandler) createUser(w http.ResponseWriter, r *http.Request) {
	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	responce, code, message, err := h.service.CreateUser(r.Context(), &req)
	if err != nil {
		functions.RespondwithError(w, code, message, err)
		return
	}

	functions.RespondwithJSON(w, code, *responce)
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

	response, code, message, err := h.service.GetUserByID(r.Context(), userIdUUID)
	if err != nil {
		functions.RespondwithError(w, code, message, err)
		return
	}

	functions.RespondwithJSON(w, http.StatusOK, *response)
}

func (h *UserHandler) getUsers(w http.ResponseWriter, r *http.Request) {

	response, code, message, err := h.service.GetUsers(r.Context())
	if err != nil {
		functions.RespondwithError(w, code, message, err)
		return
	}

	functions.RespondwithJSON(w, http.StatusOK, response)
}

func (h *UserHandler) UpdateUserStatus(w http.ResponseWriter, r *http.Request) {
	//get the user id
	userId := chi.URLParam(r, "id")
	id, err := uuid.Parse(userId)
	if err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "invalid user id", err)
		return
	}

	var req models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "invalid body", err)
		return
	}

	response, code, message, err := h.service.UpdateUserStatus(r.Context(), id, &req)
	if err != nil {
		functions.RespondwithError(w, code, message, err)
		return
	}

	functions.RespondwithJSON(w, http.StatusOK, *response)
}

func (h *UserHandler) deleteUser(w http.ResponseWriter, r *http.Request) {
	//get the user id
	userId := chi.URLParam(r, "id")
	id, err := uuid.Parse(userId)
	if err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "invalid user id", err)
		return
	}

	code, message, err := h.service.DeleteUser(r.Context(), id)
	if err != nil {
		functions.RespondwithError(w, code, message, err)
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

	response, code, message, err := h.service.GetDataUsage(r.Context(), id)
	if err != nil {
		functions.RespondwithError(w, code, message, err)
		return
	}

	functions.RespondwithJSON(w, http.StatusOK, response)
}

func (h *UserHandler) getUserAllowPools(w http.ResponseWriter, r *http.Request) {
	//get the user id
	userId := chi.URLParam(r, "id")
	id, err := uuid.Parse(userId)
	if err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "invalid user id", err)
		return
	}

	response, code, message, err := h.service.GetUserAllowPools(r.Context(), id)
	if err != nil {
		functions.RespondwithError(w, code, message, err)
		return
	}

	functions.RespondwithJSON(w, http.StatusOK, *response)
}

func (h *UserHandler) addUserAllowPool(w http.ResponseWriter, r *http.Request) {
	//get the user id
	userId := chi.URLParam(r, "id")
	id, err := uuid.Parse(userId)
	if err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "invalid user id", err)
		return
	}

	var req models.AddUserPoolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "server error", err)
		return
	}

	response, code, message, err := h.service.AddUserAllowPool(r.Context(), id, &req)
	if err != nil {
		functions.RespondwithError(w, code, message, err)
		return
	}

	functions.RespondwithJSON(w, http.StatusCreated, *response)
}

func (h *UserHandler) removeUserAllowPool(w http.ResponseWriter, r *http.Request) {
	//get the user id
	userId := chi.URLParam(r, "id")
	id, err := uuid.Parse(userId)
	if err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "invalid user id", err)
		return
	}

	var req models.DeleteUserpoolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "server error", err)
		return
	}

	code, message, err := h.service.RemoveUserAllowPool(r.Context(), id, &req)
	if err != nil {
		functions.RespondwithError(w, code, message, err)
		return
	}

	res := struct {
		Message string `json:"message"`
	}{
		Message: message,
	}

	functions.RespondwithJSON(w, code, res)
}

func (h *UserHandler) getUserIpWhitelist(w http.ResponseWriter, r *http.Request) {
	//get the user id
	userId := chi.URLParam(r, "id")
	id, err := uuid.Parse(userId)
	if err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "invalid user id", err)
		return
	}

	response, code, message, err := h.service.GetUserIpWhitelist(r.Context(), id)
	if err != nil {
		functions.RespondwithError(w, code, message, err)
		return
	}

	functions.RespondwithJSON(w, http.StatusOK, *response)

}

func (h *UserHandler) addUserIpWhitelist(w http.ResponseWriter, r *http.Request) {
	//get the user id
	userId := chi.URLParam(r, "id")
	id, err := uuid.Parse(userId)
	if err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "invalid user id", err)
		return
	}

	var req models.AddUserIpwhitelistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "server error", err)
		return
	}

	response, code, message, err := h.service.AddUserIpWhitelist(r.Context(), id, &req)
	if err != nil {
		functions.RespondwithError(w, code, message, err)
		return
	}

	functions.RespondwithJSON(w, http.StatusCreated, *response)
}

func (h *UserHandler) removeUserIpWhitelist(w http.ResponseWriter, r *http.Request) {
	//get the user id
	userId := chi.URLParam(r, "id")
	id, err := uuid.Parse(userId)
	if err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "invalid user id", err)
		return
	}

	var req models.DeleteUserIpwhitelistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "server error", err)
		return
	}

	code, message, err := h.service.RemoveUserIpWhitelist(r.Context(), id, &req)
	if err != nil {
		functions.RespondwithError(w, code, message, err)
		return
	}

	res := struct {
		Message string `json:"message"`
	}{
		Message: message,
	}

	functions.RespondwithJSON(w, http.StatusOK, res)
}

func (h *UserHandler) GenerateproxyString(w http.ResponseWriter, r *http.Request) {
	var req models.GenerateproxyStringRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "server error", err)
		return
	}

	if req.UserId == nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "server error", fmt.Errorf("user id is required"))
		return
	}
	if req.Amount == nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "server error", fmt.Errorf("amount is required"))
		return
	}
	if req.IsSticky == nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "server error", fmt.Errorf("sticky is required"))
		return
	}
	if req.CountryCode == nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "server error", fmt.Errorf("country code is required"))
		return
	}
	if req.PoolGroup == nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "server error", fmt.Errorf("pool group is required"))
		return
	}
	if req.ProxyType == nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "server error", fmt.Errorf("proxy type is required"))
		return
	}
	if req.Format == nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "server error", fmt.Errorf("format is required"))
		return
	}

	response, code, message, err := h.service.GenerateProxyString(r.Context(), &req)
	if err != nil {
		functions.RespondwithError(w, code, message, err)
		return
	}

	functions.RespondwithJSON(w, http.StatusOK, response)
}
