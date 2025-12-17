package service

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/torchlabssoftware/subnetwork_system/internal/db/repository"
	functions "github.com/torchlabssoftware/subnetwork_system/internal/server/functions"
	models "github.com/torchlabssoftware/subnetwork_system/internal/server/models"
)

type UserService interface {
	CreateUser(ctx context.Context, user *models.CreateUserRequest) (responce *models.CreateUserResponce, code int, message string, err error)
	GetUserByID(ctx context.Context, id uuid.UUID) (response *models.GetUserByIdResponce, code int, message string, err error)
	GetUsers(ctx context.Context) (response []models.GetUserByIdResponce, code int, message string, err error)
	UpdateUserStatus(ctx context.Context, id uuid.UUID, req *models.UpdateUserRequest) (response *models.UpdateUserResponce, code int, message string, err error)
	DeleteUser(ctx context.Context, id uuid.UUID) (code int, message string, err error)
	GetDataUsage(ctx context.Context, id uuid.UUID) (response []models.GetDatausageReponce, code int, message string, err error)
	GetUserAllowPools(ctx context.Context, id uuid.UUID) (response *models.GetUserPoolResponce, code int, message string, err error)
	AddUserAllowPool(ctx context.Context, id uuid.UUID, req *models.AddUserPoolRequest) (response *models.AddUserPoolResponce, code int, message string, err error)
	RemoveUserAllowPool(ctx context.Context, id uuid.UUID, req *models.DeleteUserpoolRequest) (code int, message string, err error)
	GetUserIpWhitelist(ctx context.Context, id uuid.UUID) (response *models.GetUserIpwhitelistResponce, code int, message string, err error)
	AddUserIpWhitelist(ctx context.Context, id uuid.UUID, req *models.AddUserIpwhitelistRequest) (response *models.AddUserIpwhitelistResponce, code int, message string, err error)
	RemoveUserIpWhitelist(ctx context.Context, id uuid.UUID, req *models.DeleteUserIpwhitelistRequest) (code int, message string, err error)
	GenerateProxyString(ctx context.Context, req *models.GenerateproxyStringRequest) (response []string, code int, message string, err error)
}

type userService struct {
	queries *repository.Queries
	db      *sql.DB
}

func NewUserService(q *repository.Queries, db *sql.DB) UserService {
	return &userService{queries: q, db: db}
}

func (u *userService) CreateUser(context context.Context, req *models.CreateUserRequest) (responce *models.CreateUserResponce, code int, message string, err error) {
	//begin transaction
	ctx, err := u.db.BeginTx(context, nil)
	if err != nil {
		return nil, http.StatusInternalServerError, "failed to create user", err
	}
	defer func() {
		_ = ctx.Rollback()
	}()

	qtx := u.queries.WithTx(ctx)

	//insert user data
	createUserParams := repository.CreateUserParams{
		Username: uuid.New().String()[:8],
		Password: uuid.New().String()[:8],
	}

	user, err := qtx.CreateUser(context, createUserParams)
	if err != nil {
		return nil, http.StatusInternalServerError, "failed to create user", err
	}

	//insert pool data
	var addedPools repository.AddUserPoolsByPoolTagsRow
	if req.AllowPools != nil && len(*req.AllowPools) > 0 {
		pools_tags := []string{}
		dataLimit := []int64{}

		for _, pool := range *req.AllowPools {
			pools_tags = append(pools_tags, pool.Pool)
			dataLimit = append(dataLimit, pool.DataLimit)
		}

		poolArgs := repository.AddUserPoolsByPoolTagsParams{
			UserID:     user.ID,
			Tags:       pools_tags,
			DataLimits: dataLimit,
		}

		addedPools, err = qtx.AddUserPoolsByPoolTags(context, poolArgs)
		if err != nil {
			return nil, http.StatusInternalServerError, "failed to create user", err
		}
	}

	//insert ipwhilist data
	var ipWhitelist repository.InsertUserIpwhitelistRow
	if req.IpWhiteList != nil && len(*req.IpWhiteList) > 0 {

		userIpWhitelistParams := repository.InsertUserIpwhitelistParams{
			UserID:      user.ID,
			IpWhitelist: *req.IpWhiteList,
		}

		ipWhitelist, err = qtx.InsertUserIpwhitelist(context, userIpWhitelistParams)
		if err != nil {
			return nil, http.StatusInternalServerError, "failed to create user", err
		}

	}

	if err := ctx.Commit(); err != nil {
		return nil, http.StatusInternalServerError, "failed to create user", err
	}

	responce = &models.CreateUserResponce{
		Id:          user.ID,
		Username:    user.Username,
		Password:    user.Password,
		Status:      user.Status,
		IpWhitelist: ipWhitelist.IpWhitelist,
		AllowPools:  addedPools.InsertedTags,
		Created_at:  user.CreatedAt,
		Updated_at:  user.UpdatedAt,
	}

	return responce, http.StatusCreated, "user created", nil
}

func (u *userService) GetUserByID(ctx context.Context, id uuid.UUID) (response *models.GetUserByIdResponce, code int, message string, err error) {
	user, err := u.queries.GetUserbyId(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, http.StatusNotFound, "user not found", err
		}
		return nil, http.StatusInternalServerError, "cant get user by id", err
	}

	response = &models.GetUserByIdResponce{
		Id:          user.ID,
		Username:    user.Username,
		Password:    user.Password,
		Status:      user.Status,
		IpWhitelist: user.IpWhitelist,
		UserPool:    user.Pools,
		Created_at:  user.CreatedAt,
		Updated_at:  user.UpdatedAt,
	}

	return response, http.StatusOK, "", nil
}

func (u *userService) GetUsers(ctx context.Context) (response []models.GetUserByIdResponce, code int, message string, err error) {
	users, err := u.queries.GetAllusers(ctx)
	if err != nil {
		return nil, http.StatusInternalServerError, "cant get users", err
	}

	response = []models.GetUserByIdResponce{}
	for _, user := range users {
		response = append(response, models.GetUserByIdResponce{
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
	return response, http.StatusOK, "", nil
}

func (u *userService) UpdateUserStatus(ctx context.Context, id uuid.UUID, req *models.UpdateUserRequest) (response *models.UpdateUserResponce, code int, message string, err error) {
	params := repository.UpdateUserParams{
		ID: id,
	}

	if req.Status != nil && *req.Status != "" {
		params.Status = sql.NullString{String: *req.Status, Valid: true}
	}

	user, err := u.queries.UpdateUser(ctx, params)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, http.StatusNotFound, "user not found", err
		}
		return nil, http.StatusInternalServerError, "server error", err
	}

	response = &models.UpdateUserResponce{
		Id:        user.ID,
		Status:    user.Status,
		UpdatedAt: user.UpdatedAt,
	}

	return response, http.StatusOK, "", nil
}

func (u *userService) DeleteUser(ctx context.Context, id uuid.UUID) (code int, message string, err error) {
	err = u.queries.DeleteUser(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return http.StatusNotFound, "user not found", err
		}
		return http.StatusInternalServerError, "server error", err
	}
	return http.StatusOK, "user deleted", nil
}

func (u *userService) GetDataUsage(ctx context.Context, id uuid.UUID) (response []models.GetDatausageReponce, code int, message string, err error) {
	dataUsages, err := u.queries.GetDatausageById(ctx, id)
	if err != nil {
		return nil, http.StatusInternalServerError, "server error", err
	}

	response = []models.GetDatausageReponce{}
	for _, dataUsage := range dataUsages {
		response = append(response, models.GetDatausageReponce{
			DataLimit: dataUsage.DataLimit,
			DataUsage: dataUsage.DataUsage,
			PoolTag:   dataUsage.PoolTag,
		})
	}

	return response, http.StatusOK, "", nil
}

func (u *userService) GetUserAllowPools(ctx context.Context, id uuid.UUID) (response *models.GetUserPoolResponce, code int, message string, err error) {
	userPool, err := u.queries.GetUserPoolsByUserId(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, http.StatusNotFound, "pools not found", err
		}
		return nil, http.StatusInternalServerError, "server error", err
	}

	poolDataStat := []string{}

	poolDataStat = append(poolDataStat, userPool.PoolTags...)

	response = &models.GetUserPoolResponce{
		Pools: poolDataStat,
	}

	return response, http.StatusOK, "", nil
}

func (u *userService) AddUserAllowPool(ctx context.Context, id uuid.UUID, req *models.AddUserPoolRequest) (response *models.AddUserPoolResponce, code int, message string, err error) {
	tags := []string{}
	dataLimits := []int64{}

	for _, pool := range req.UserPool {
		tags = append(tags, pool.Pool)
		dataLimits = append(dataLimits, pool.DataLimit)
	}

	args := repository.AddUserPoolsByPoolTagsParams{
		UserID:     id,
		Tags:       tags,
		DataLimits: dataLimits,
	}

	pool, err := u.queries.AddUserPoolsByPoolTags(ctx, args)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, http.StatusNotFound, "Nothing added to the database", err
		}
		return nil, http.StatusInternalServerError, "server error", err
	}

	userPool := []models.PoolDataStat{}

	for i, d := range pool.InsertedDataLimits {
		userPool = append(userPool, models.PoolDataStat{
			Pool:      pool.InsertedTags[i],
			DataLimit: d,
		})
	}

	response = &models.AddUserPoolResponce{
		UserPool: userPool,
	}

	return response, http.StatusCreated, "", nil
}

func (u *userService) RemoveUserAllowPool(ctx context.Context, id uuid.UUID, req *models.DeleteUserpoolRequest) (code int, message string, err error) {
	args := repository.DeleteUserPoolsByTagsParams{
		UserID:  id,
		Column2: req.UserPool,
	}

	res, err := u.queries.DeleteUserPoolsByTags(ctx, args)
	if err != nil {
		return http.StatusInternalServerError, "server error", err
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return http.StatusNotFound, "Nothing deleted from the database", nil
	}

	return http.StatusOK, "pool removed", nil
}

func (u *userService) GetUserIpWhitelist(ctx context.Context, id uuid.UUID) (response *models.GetUserIpwhitelistResponce, code int, message string, err error) {
	userPool, err := u.queries.GetUserIpwhitelistByUserId(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, http.StatusNotFound, "user ip whitelist not found", err
		}
		return nil, http.StatusInternalServerError, "server error", err
	}

	response = &models.GetUserIpwhitelistResponce{
		IpWhitelist: userPool,
	}

	return response, http.StatusOK, "", nil
}

func (u *userService) AddUserIpWhitelist(ctx context.Context, id uuid.UUID, req *models.AddUserIpwhitelistRequest) (response *models.AddUserIpwhitelistResponce, code int, message string, err error) {
	args := repository.InsertUserIpwhitelistParams{
		UserID:      id,
		IpWhitelist: req.IpWhitelist,
	}

	iplist, err := u.queries.InsertUserIpwhitelist(ctx, args)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, http.StatusNotFound, "nothing added to the database", err
		}
		return nil, http.StatusInternalServerError, "server error", err
	}

	response = &models.AddUserIpwhitelistResponce{
		IpWhitelist: iplist.IpWhitelist,
	}

	return response, http.StatusCreated, "", nil
}

func (u *userService) RemoveUserIpWhitelist(ctx context.Context, id uuid.UUID, req *models.DeleteUserIpwhitelistRequest) (code int, message string, err error) {
	arg := repository.DeleteUserIpwhitelistParams{
		UserID:  id,
		Column2: req.IpCidr,
	}

	res, err := u.queries.DeleteUserIpwhitelist(ctx, arg)
	if err != nil {
		return http.StatusInternalServerError, "server error", err
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return http.StatusNotFound, "Nothing deleted from the database", nil
	}

	return http.StatusOK, "ip whitelist removed", nil
}

func (u *userService) GenerateProxyString(ctx context.Context, req *models.GenerateproxyStringRequest) (response []string, code int, message string, err error) {
	tag := *req.PoolGroup + *req.ProxyType + "%"

	data, err := u.queries.GenerateproxyString(ctx, repository.GenerateproxyStringParams{
		Code:   *req.CountryCode,
		Tag:    tag,
		UserID: *req.UserId,
	})
	if err != nil {
		return nil, http.StatusInternalServerError, "server error", err
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
	return res, http.StatusOK, "", nil
}
