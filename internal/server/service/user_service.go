package service

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/google/uuid"
	"github.com/torchlabssoftware/subnetwork_system/internal/db/repository"
	models "github.com/torchlabssoftware/subnetwork_system/internal/server/models"
)

type UserService interface {
	CreateUser(ctx context.Context, user *models.CreateUserRequest) (responce *models.CreateUserResponce, code int, message string, err error)
	GetUserByID(ctx context.Context, id uuid.UUID) (response *models.GetUserByIdResponce, code int, message string, err error)
	GetUsers(ctx context.Context) (response []models.GetUserByIdResponce, code int, message string, err error)
	UpdateUserStatus(ctx context.Context, id uuid.UUID, req *models.UpdateUserRequest) (response *models.UpdateUserResponce, code int, message string, err error)
	DeleteUser(ctx context.Context, id uuid.UUID) (code int, message string, err error)
	GetDataUsage(ctx context.Context, id uuid.UUID) (response []models.GetDatausageReponce, code int, message string, err error)
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
