package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/torchlabssoftware/subnetwork_system/internal/db/repository"
	functions "github.com/torchlabssoftware/subnetwork_system/internal/server/functions"
	middleware "github.com/torchlabssoftware/subnetwork_system/internal/server/middleware"
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

	r.Use(middleware.AdminAuthentication)

	r.Post("/", h.createUser)
	r.Get("/", h.getUsers)
	r.Get("/{id}", h.getUserbyId)
	r.Patch("/{id}", h.updateUser)
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

	//insert user data
	createUserParams := repository.CreateUserParams{
		Username: uuid.New().String()[:8],
		Password: uuid.New().String()[:8],
	}

	user, err := qtx.CreateUser(r.Context(), createUserParams)
	if err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "failed to create user", err)
		return
	}

	log.Println("uid: ", user.ID)

	//insert allow_pool data
	var allowPools []server.PoolDataStat
	if req.AllowPools != nil || len(*req.AllowPools) > 0 {
		allowPools = *req.AllowPools
	}

	pools := []string{}
	dataLimit := []int64{}

	for _, pool := range allowPools {
		pools = append(pools, pool.Pool)
		dataLimit = append(dataLimit, pool.DataLimit)
	}

	poolArgs := repository.AddUserPoolsByPoolTagsParams{
		UserID:  user.ID,
		Column2: pools,
		Column3: dataLimit,
	}

	addedPools, err := qtx.AddUserPoolsByPoolTags(r.Context(), poolArgs)
	if err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "failed to create user", err)
		return
	}

	log.Println("addedPools: ", len(addedPools.InsertedTags))

	//insert ipwhilist data
	var ipWhitelist []string
	if req.IpWhiteList != nil || len(*req.IpWhiteList) > 0 {
		ipWhitelist = *req.IpWhiteList
	}

	userIpWhitelistParams := repository.InsertUserIpwhitelistParams{
		UserID:  user.ID,
		Column2: ipWhitelist,
	}

	ip, err := qtx.InsertUserIpwhitelist(r.Context(), userIpWhitelistParams)
	if err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "failed to create user", err)
		return
	}

	log.Println(len(ip))

	if err := ctx.Commit(); err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "failed to create user", err)
		return
	}

	responce := server.CreateUserResponce{
		Id:          user.ID,
		Username:    user.Username,
		Password:    user.Password,
		IpWhitelist: ipWhitelist,
		AllowPools:  addedPools.InsertedTags,
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
		Username:    user.Username,
		Password:    user.Password,
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
			Username:    user.Username,
			Password:    user.Password,
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

	dataUsages, err := h.queries.GetDatausageById(r.Context(), id)
	if err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "server error", err)
		return
	}

	res := []server.GetDatausageReponce{}

	for _, dataUsage := range dataUsages {
		res = append(res, server.GetDatausageReponce{
			DataLimit: dataUsage.DataLimit,
			DataUsage: dataUsage.DataUsage,
			PoolTag:   dataUsage.PoolTag,
		})
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

	poolDataStat := []server.PoolDataStat{}

	for i := range userPool.PoolIds {
		poolDataStat = append(poolDataStat, server.PoolDataStat{
			Pool:      userPool.PoolIds[i],
			DataLimit: userPool.DataLimits[i],
			DataUsage: userPool.DataUsages[i],
		})
	}

	resp := server.GetUserPoolResponce{
		UserId: userPool.ID,
		Pools:  poolDataStat,
	}

	functions.RespondwithJSON(w, http.StatusOK, resp)

}

func (h *UserHandler) addUserAllowPool(w http.ResponseWriter, r *http.Request) {
	//get the user id
	userId := chi.URLParam(r, "id")
	id, err := uuid.Parse(userId)
	if err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "invalid user id", err)
		return
	}

	var req server.AddUserPoolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "server error", err)
		return
	}

	tags := []string{}
	dataLimits := []int64{}

	for _, pool := range req.UserPool {
		tags = append(tags, pool.Pool)
		dataLimits = append(dataLimits, pool.DataLimit)
	}

	args := repository.AddUserPoolsByPoolTagsParams{
		UserID:  id,
		Column2: tags,
		Column3: dataLimits,
	}

	pool, err := h.queries.AddUserPoolsByPoolTags(r.Context(), args)
	if err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "server error", err)
		return
	}

	userPool := []server.PoolDataStat{}

	for i, d := range pool.InsertedDataLimits {
		userPool = append(userPool, server.PoolDataStat{
			Pool:      pool.InsertedTags[i],
			DataLimit: d,
		})
	}

	res := server.AddUserPoolResponce{
		UserId:   id,
		UserPool: userPool,
	}

	functions.RespondwithJSON(w, http.StatusCreated, res)
}

func (h *UserHandler) removeUserAllowPool(w http.ResponseWriter, r *http.Request) {
	//get the user id
	userId := chi.URLParam(r, "id")
	id, err := uuid.Parse(userId)
	if err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "invalid user id", err)
		return
	}

	var req server.DeleteUserpoolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "server error", err)
		return
	}

	arg := repository.DeleteUserPoolsByTagsParams{
		UserID:  id,
		Column2: req.UserPool,
	}

	err = h.queries.DeleteUserPoolsByTags(r.Context(), arg)
	if err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "server error", err)
		return
	}

	functions.RespondwithJSON(w, http.StatusOK, nil)
}

func (h *UserHandler) getUserIpWhitelist(w http.ResponseWriter, r *http.Request) {
	//get the user id
	userId := chi.URLParam(r, "id")
	id, err := uuid.Parse(userId)
	if err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "invalid user id", err)
		return
	}

	userPool, err := h.queries.GetUserIpwhitelistByUserId(r.Context(), id)
	if err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "server error", err)
		return
	}

	resp := server.GetUserIpwhitelistResponce{
		UserId:      userPool.UserID,
		IpWhitelist: userPool.IpList,
	}

	functions.RespondwithJSON(w, http.StatusOK, resp)

}

func (h *UserHandler) addUserIpWhitelist(w http.ResponseWriter, r *http.Request) {
	//get the user id
	userId := chi.URLParam(r, "id")
	id, err := uuid.Parse(userId)
	if err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "invalid user id", err)
		return
	}

	var req server.AddUserIpwhitelistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "server error", err)
		return
	}

	args := repository.InsertUserIpwhitelistParams{
		UserID:  id,
		Column2: req.IpWhitelist,
	}

	_, err = h.queries.InsertUserIpwhitelist(r.Context(), args)
	if err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "server error", err)
		return
	}

	res := server.AddUserIpwhitelistResponce{
		UserId:      id,
		IpWhitelist: req.IpWhitelist,
	}

	functions.RespondwithJSON(w, http.StatusCreated, res)
}

func (h *UserHandler) removeUserIpWhitelist(w http.ResponseWriter, r *http.Request) {
	//get the user id
	userId := chi.URLParam(r, "id")
	id, err := uuid.Parse(userId)
	if err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "invalid user id", err)
		return
	}

	var req server.DeleteUserIpwhitelistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "server error", err)
		return
	}

	arg := repository.DeleteUserIpwhitelistParams{
		UserID:  id,
		Column2: req.IpCidr,
	}

	err = h.queries.DeleteUserIpwhitelist(r.Context(), arg)
	if err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "server error", err)
		return
	}

	functions.RespondwithJSON(w, http.StatusOK, nil)
}

func (h *UserHandler) GenerateproxyString(w http.ResponseWriter, r *http.Request) {
	var req server.GenerateproxyStringRequest
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

	tag := *req.PoolGroup + *req.ProxyType + "%"

	data, err := h.queries.GenerateproxyString(r.Context(), repository.GenerateproxyStringParams{
		Code:   *req.CountryCode,
		Tag:    tag,
		UserID: *req.UserId,
	})
	if err != nil {
		functions.RespondwithError(w, http.StatusInternalServerError, "server error", err)
		return
	}

	userName := data.Username
	password := data.Password
	subdomain := data.Subdomain
	port := data.Port

	res := []string{}

	for i := 0; i < *req.Amount; i++ {
		config := functions.GenerateproxyString(*req.PoolGroup, *req.CountryCode, *req.IsSticky)
		switch *req.Format {
		case "ip:port:user:pass":
			proxyString := fmt.Sprintf("%s"+"upstream-y.com"+":%d:%s:%s%s", subdomain, port, userName, password, config)
			res = append(res, proxyString)
		case "user:pass:ip:port":
			proxyString := fmt.Sprintf("%s:%s%s:%s"+"upstream-y.com"+":%d", userName, password, config, subdomain, port)
			res = append(res, proxyString)
		case "user:pass@ip:port":
			proxyString := fmt.Sprintf("%s:%s%s@%s"+"upstream-y.com"+":%d", userName, password, config, subdomain, port)
			res = append(res, proxyString)
		}
	}

	functions.RespondwithJSON(w, http.StatusOK, res)
}
