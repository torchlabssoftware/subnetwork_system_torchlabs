package unit

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	handlers "github.com/torchlabssoftware/subnetwork_system/internal/server/handlers"
	models "github.com/torchlabssoftware/subnetwork_system/internal/server/models"
	"github.com/torchlabssoftware/subnetwork_system/tests/mocks"
)

func TestCreateUser_Handler(t *testing.T) {

	mockService := new(mocks.MockUserService)

	h := handlers.NewUserHandler(mockService)

	router := h.TestRoutes()

	t.Run("Success Case", func(t *testing.T) {
		// Arrange
		reqBody := models.CreateUserRequest{
			IpWhiteList: &[]string{"192.168.1.23", "192.168.1.53", "192.168.15.123"},
			AllowPools: &[]models.PoolDataStat{
				{
					Pool:      "netnutusa",
					DataLimit: 1000,
				},
				{
					Pool:      "iproyalusa",
					DataLimit: 1000,
				},
				{
					Pool:      "geonodeusa",
					DataLimit: 1000,
				},
				{
					Pool:      "netnuteu",
					DataLimit: 1000,
				},
				{
					Pool:      "netnutasia",
					DataLimit: 1000,
				},
			},
		}
		expectedResp := &models.CreateUserResponce{
			Id:          uuid.New(),
			Username:    uuid.New().String()[:8],
			Password:    uuid.New().String()[:8],
			Status:      "active",
			IpWhitelist: []string{"192.168.1.23", "192.168.1.53", "192.168.15.123"},
			AllowPools: []string{
				"netnutusa",
				"iproyalusa",
				"geonodeusa",
				"netnuteu",
				"netnutasia",
			},
			Created_at: time.Now(),
			Updated_at: time.Now(),
		}

		// Expect the Service to be called
		mockService.On("CreateUser", mock.Anything, mock.Anything).
			Return(expectedResp, http.StatusCreated, "user created", nil).
			Once()

		// Act
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "ApiKey K9@pL2$mQ#z9jR!vN7kY&wB6tG*hF5dC")
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusCreated, rr.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("Service Error Case", func(t *testing.T) {
		// Arrange
		mockService.On("CreateUser", mock.Anything, mock.Anything).
			Return(nil, http.StatusInternalServerError, "db error", errors.New("boom")).
			Once()

		// Act
		req, _ := http.NewRequest("POST", "/", bytes.NewBuffer([]byte("{}")))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		mockService.AssertExpectations(t)
	})
}
