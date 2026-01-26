package e2e

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	models "github.com/torchlabssoftware/subnetwork_system/internal/server/models"
	"github.com/torchlabssoftware/subnetwork_system/tests/e2e/helpers"
)

// Analytics API Tests

func TestE2E_GetUserUsage(t *testing.T) {
	client := GetAdminClient()
	createReq := models.CreateUserRequest{}
	createResp := client.Post(t, "/admin/users/", createReq)
	createResp.RequireStatus(t, http.StatusCreated)
	var user models.CreateUserResponce
	createResp.ParseJSON(t, &user)
	resp := client.Get(t, "/admin/analytics/user/"+user.Id.String()+"/usage?from=2024-01-01&to=2025-12-31")
	resp.RequireStatus(t, http.StatusOK)
	t.Logf("User usage response: %s", resp.String())
}

func TestE2E_GetWorkerHealth(t *testing.T) {
	adminClient := GetAdminClient()
	poolId := createTestPoolForWorker(t, adminClient)
	poolUUID, _ := uuid.Parse(poolId)
	createReq := models.AddWorkerRequest{
		RegionName: helpers.Ptr("North America"),
		IPAddress:  helpers.Ptr("192.168.50.1"),
		Port:       helpers.Ptr(int32(8080)),
		PoolId:     helpers.Ptr(poolUUID),
	}
	createResp := adminClient.Post(t, "/admin/worker/", createReq)
	createResp.RequireStatus(t, http.StatusOK)
	var worker models.AddWorkerResponse
	createResp.ParseJSON(t, &worker)
	resp := adminClient.Get(t, "/admin/analytics/worker/"+worker.ID+"/health?from=2024-01-01&to=2025-12-31")
	resp.RequireStatus(t, http.StatusOK)
	t.Logf("Worker health response: %s", resp.String())
}

func TestE2E_GetUserWebsiteAccess(t *testing.T) {
	client := GetAdminClient()
	createReq := models.CreateUserRequest{}
	createResp := client.Post(t, "/admin/users/", createReq)
	createResp.RequireStatus(t, http.StatusCreated)
	var user models.CreateUserResponce
	createResp.ParseJSON(t, &user)
	resp := client.Get(t, "/admin/analytics/user/"+user.Id.String()+"/website-access?from=2024-01-01&to=2025-12-31")
	resp.RequireStatus(t, http.StatusOK)
	t.Logf("Website access response: %s", resp.String())
}

func TestE2E_GetUserUsage_InvalidDates(t *testing.T) {
	client := GetAdminClient()
	createReq := models.CreateUserRequest{}
	createResp := client.Post(t, "/admin/users/", createReq)
	createResp.RequireStatus(t, http.StatusCreated)
	var user models.CreateUserResponce
	createResp.ParseJSON(t, &user)
	resp := client.Get(t, "/admin/analytics/user/"+user.Id.String()+"/usage?from=invalid&to=also-invalid")
	resp.RequireStatus(t, http.StatusOK)
}
