package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	models "github.com/torchlabssoftware/subnetwork_system/internal/server/models"
)

type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) CreateUser(ctx context.Context, user *models.CreateUserRequest) (*models.CreateUserResponce, int, string, error) {
	args := m.Called(ctx, user)

	var resp *models.CreateUserResponce
	if args.Get(0) != nil {
		resp = args.Get(0).(*models.CreateUserResponce)
	}
	return resp, args.Int(1), args.String(2), args.Error(3)
}

func (m *MockUserService) GetUserByID(ctx context.Context, id uuid.UUID) (*models.GetUserByIdResponce, int, string, error) {
	args := m.Called(ctx, id)

	var resp *models.GetUserByIdResponce
	if args.Get(0) != nil {
		resp = args.Get(0).(*models.GetUserByIdResponce)
	}
	return resp, args.Int(1), args.String(2), args.Error(3)
}

func (m *MockUserService) GetUsers(ctx context.Context) ([]models.GetUserByIdResponce, int, string, error) {
	args := m.Called(ctx)

	var resp []models.GetUserByIdResponce
	if args.Get(0) != nil {
		resp = args.Get(0).([]models.GetUserByIdResponce)
	}
	return resp, args.Int(1), args.String(2), args.Error(3)
}

func (m *MockUserService) UpdateUserStatus(ctx context.Context, id uuid.UUID, req *models.UpdateUserRequest) (*models.UpdateUserResponce, int, string, error) {
	args := m.Called(ctx, id, req)

	var resp *models.UpdateUserResponce
	if args.Get(0) != nil {
		resp = args.Get(0).(*models.UpdateUserResponce)
	}
	return resp, args.Int(1), args.String(2), args.Error(3)
}

func (m *MockUserService) DeleteUser(ctx context.Context, id uuid.UUID) (int, string, error) {
	args := m.Called(ctx, id)
	return args.Int(0), args.String(1), args.Error(2)
}

func (m *MockUserService) GetDataUsage(ctx context.Context, id uuid.UUID) ([]models.GetDatausageReponce, int, string, error) {
	args := m.Called(ctx, id)

	var resp []models.GetDatausageReponce
	if args.Get(0) != nil {
		resp = args.Get(0).([]models.GetDatausageReponce)
	}
	return resp, args.Int(1), args.String(2), args.Error(3)
}

func (m *MockUserService) GetUserAllowPools(ctx context.Context, id uuid.UUID) (*models.GetUserPoolResponce, int, string, error) {
	args := m.Called(ctx, id)

	var resp *models.GetUserPoolResponce
	if args.Get(0) != nil {
		resp = args.Get(0).(*models.GetUserPoolResponce)
	}
	return resp, args.Int(1), args.String(2), args.Error(3)
}

func (m *MockUserService) AddUserAllowPool(ctx context.Context, id uuid.UUID, req *models.AddUserPoolRequest) (*models.AddUserPoolResponce, int, string, error) {
	args := m.Called(ctx, id, req)

	var resp *models.AddUserPoolResponce
	if args.Get(0) != nil {
		resp = args.Get(0).(*models.AddUserPoolResponce)
	}
	return resp, args.Int(1), args.String(2), args.Error(3)
}

func (m *MockUserService) RemoveUserAllowPool(ctx context.Context, id uuid.UUID, req *models.DeleteUserpoolRequest) (int, string, error) {
	args := m.Called(ctx, id, req)
	return args.Int(0), args.String(1), args.Error(2)
}

func (m *MockUserService) GetUserIpWhitelist(ctx context.Context, id uuid.UUID) (*models.GetUserIpwhitelistResponce, int, string, error) {
	args := m.Called(ctx, id)

	var resp *models.GetUserIpwhitelistResponce
	if args.Get(0) != nil {
		resp = args.Get(0).(*models.GetUserIpwhitelistResponce)
	}
	return resp, args.Int(1), args.String(2), args.Error(3)
}

func (m *MockUserService) AddUserIpWhitelist(ctx context.Context, id uuid.UUID, req *models.AddUserIpwhitelistRequest) (*models.AddUserIpwhitelistResponce, int, string, error) {
	args := m.Called(ctx, id, req)

	var resp *models.AddUserIpwhitelistResponce
	if args.Get(0) != nil {
		resp = args.Get(0).(*models.AddUserIpwhitelistResponce)
	}
	return resp, args.Int(1), args.String(2), args.Error(3)
}

func (m *MockUserService) RemoveUserIpWhitelist(ctx context.Context, id uuid.UUID, req *models.DeleteUserIpwhitelistRequest) (int, string, error) {
	args := m.Called(ctx, id, req)
	return args.Int(0), args.String(1), args.Error(2)
}

func (m *MockUserService) GenerateProxyString(ctx context.Context, req *models.GenerateproxyStringRequest) ([]string, int, string, error) {
	args := m.Called(ctx, req)

	var resp []string
	if args.Get(0) != nil {
		resp = args.Get(0).([]string)
	}
	return resp, args.Int(1), args.String(2), args.Error(3)
}
