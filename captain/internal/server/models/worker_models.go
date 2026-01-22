package server

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/torchlabssoftware/subnetwork_system/internal/db/repository"
)

type WebsocketManagerInterface interface {
	NewOTP(workerId *uuid.UUID) string
	VerifyOTP(otp string) (bool, uuid.UUID)
	ServeWS(w http.ResponseWriter, r *http.Request, workerID uuid.UUID, workerName string, poolId uuid.UUID)
	NotifyUserChange(username string)
	NotifyPoolChange(poolId uuid.UUID)
	SetAnalyticsandQueries(queries *repository.Queries, analytics AnalyticsService)
}

type AddWorkerRequest struct {
	RegionName *string    `json:"region_name"`
	IPAddress  *string    `json:"ip_address"`
	Port       *int32     `json:"port"`
	PoolId     *uuid.UUID `json:"pool_id"`
}

type AddWorkerResponse struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	RegionName string    `json:"region_name"`
	IpAddress  string    `json:"ip_address"`
	Status     string    `json:"status"`
	LastSeen   string    `json:"last_seen"`
	Port       int32     `json:"port"`
	PoolId     uuid.UUID `json:"pool_id"`
	CreatedAt  string    `json:"created_at"`
	Domains    []string  `json:"domains,omitempty"`
}

type AddWorkerDomainRequest struct {
	Domain []string `json:"domains"`
}

type DeleteWorkerDomainRequest struct {
	Domain []string `json:"domains"`
}

type WorkerLoginRequest struct {
	WorkerId *uuid.UUID `json:"worker_id"`
}

type WorkerLoginResponce struct {
	Otp string `json:"otp"`
}
