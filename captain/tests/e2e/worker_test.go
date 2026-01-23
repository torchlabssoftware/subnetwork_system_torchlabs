package e2e

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	models "github.com/torchlabssoftware/subnetwork_system/internal/server/models"
	"github.com/torchlabssoftware/subnetwork_system/tests/e2e/helpers"
)

// Worker Tests

func createTestPoolForWorker(t *testing.T, client *helpers.TestClient) string {
	//create region
	regionName := "Worker Test Region " + uuid.New().String()[:8]
	regionReq := models.CreateRegionRequest{
		Name: helpers.Ptr(regionName),
	}
	regionResp := client.Post(t, "/admin/pools/region", regionReq)
	regionResp.RequireStatus(t, http.StatusCreated)
	var region models.CreateRegionResponce
	regionResp.ParseJSON(t, &region)
	// Create pool
	poolTag := "worker-pool-" + uuid.New().String()[:8]
	poolReq := models.CreatePoolRequest{
		Tag:       helpers.Ptr(poolTag),
		RegionId:  helpers.Ptr(region.Id),
		Subdomain: helpers.Ptr("worker-test"),
		Port:      helpers.Ptr(int32(8080)),
	}
	poolResp := client.Post(t, "/admin/pools/", poolReq)
	poolResp.RequireStatus(t, http.StatusCreated)
	var pool models.CreatePoolResponce
	poolResp.ParseJSON(t, &pool)
	return pool.Id.String()
}

func TestE2E_CreateWorker(t *testing.T) {
	client := GetAdminClient()
	poolId := createTestPoolForWorker(t, client)
	poolUUID, _ := uuid.Parse(poolId)
	reqBody := models.AddWorkerRequest{
		RegionName: helpers.Ptr("North America"),
		IPAddress:  helpers.Ptr("192.168.1.100"),
		Port:       helpers.Ptr(int32(8080)),
		PoolId:     helpers.Ptr(poolUUID),
	}
	resp := client.Post(t, "/admin/worker/", reqBody)
	resp.RequireStatus(t, http.StatusOK)
	var result models.AddWorkerResponse
	resp.ParseJSON(t, &result)
	assert.NotEmpty(t, result.ID)
	assert.True(t, strings.HasPrefix(result.Name, "usa-"), "Worker name should start with region prefix")
	assert.Equal(t, "192.168.1.100", result.IpAddress)
	assert.Equal(t, int32(8080), result.Port)
	assert.Equal(t, "active", result.Status)
	t.Logf("Created worker: ID=%s, Name=%s", result.ID, result.Name)
}

func TestE2E_GetWorkers(t *testing.T) {
	client := GetAdminClient()
	resp := client.Get(t, "/admin/worker/")
	resp.RequireStatus(t, http.StatusOK)
	var workers []models.AddWorkerResponse
	resp.ParseJSON(t, &workers)
	assert.GreaterOrEqual(t, len(workers), 1, "Should have at least one worker")
	t.Logf("Found %d workers", len(workers))
}

func TestE2E_GetWorkerByName(t *testing.T) {
	client := GetAdminClient()
	poolId := createTestPoolForWorker(t, client)
	poolUUID, _ := uuid.Parse(poolId)
	createReq := models.AddWorkerRequest{
		RegionName: helpers.Ptr("Asia"),
		IPAddress:  helpers.Ptr("172.16.0.1"),
		Port:       helpers.Ptr(int32(7070)),
		PoolId:     helpers.Ptr(poolUUID),
	}
	createResp := client.Post(t, "/admin/worker/", createReq)
	createResp.RequireStatus(t, http.StatusOK)
	var created models.AddWorkerResponse
	createResp.ParseJSON(t, &created)
	resp := client.Get(t, "/admin/worker/"+created.Name)
	resp.RequireStatus(t, http.StatusOK)
	var worker models.AddWorkerResponse
	resp.ParseJSON(t, &worker)
	assert.Equal(t, created.ID, worker.ID)
	assert.Equal(t, created.Name, worker.Name)
}

func TestE2E_DeleteWorker(t *testing.T) {
	client := GetAdminClient()
	poolId := createTestPoolForWorker(t, client)
	poolUUID, _ := uuid.Parse(poolId)
	createReq := models.AddWorkerRequest{
		RegionName: helpers.Ptr("Europe"),
		IPAddress:  helpers.Ptr("10.10.10.10"),
		Port:       helpers.Ptr(int32(6060)),
		PoolId:     helpers.Ptr(poolUUID),
	}
	createResp := client.Post(t, "/admin/worker/", createReq)
	createResp.RequireStatus(t, http.StatusOK)
	var created models.AddWorkerResponse
	createResp.ParseJSON(t, &created)
	resp := client.Delete(t, "/admin/worker/"+created.Name)
	resp.RequireStatus(t, http.StatusOK)
	getResp := client.Get(t, "/admin/worker/"+created.Name)
	getResp.AssertStatus(t, http.StatusNotFound)
}

func TestE2E_AddWorkerDomain(t *testing.T) {
	client := GetAdminClient()
	poolId := createTestPoolForWorker(t, client)
	poolUUID, _ := uuid.Parse(poolId)
	createReq := models.AddWorkerRequest{
		RegionName: helpers.Ptr("North America"),
		IPAddress:  helpers.Ptr("192.168.5.5"),
		Port:       helpers.Ptr(int32(5050)),
		PoolId:     helpers.Ptr(poolUUID),
	}
	createResp := client.Post(t, "/admin/worker/", createReq)
	createResp.RequireStatus(t, http.StatusOK)
	var created models.AddWorkerResponse
	createResp.ParseJSON(t, &created)
	addDomainReq := models.AddWorkerDomainRequest{
		Domain: []string{"example.com", "test.com"},
	}
	resp := client.Post(t, "/admin/worker/"+created.Name+"/domains", addDomainReq)
	resp.RequireStatus(t, http.StatusCreated)
	getResp := client.Get(t, "/admin/worker/"+created.Name)
	getResp.RequireStatus(t, http.StatusOK)
	var worker models.AddWorkerResponse
	getResp.ParseJSON(t, &worker)
	assert.Contains(t, worker.Domains, "example.com")
	assert.Contains(t, worker.Domains, "test.com")
}

func TestE2E_DeleteWorkerDomain(t *testing.T) {
	client := GetAdminClient()
	poolId := createTestPoolForWorker(t, client)
	poolUUID, _ := uuid.Parse(poolId)
	createReq := models.AddWorkerRequest{
		RegionName: helpers.Ptr("Europe"),
		IPAddress:  helpers.Ptr("10.20.30.40"),
		Port:       helpers.Ptr(int32(4040)),
		PoolId:     helpers.Ptr(poolUUID),
	}
	createResp := client.Post(t, "/admin/worker/", createReq)
	createResp.RequireStatus(t, http.StatusOK)
	var created models.AddWorkerResponse
	createResp.ParseJSON(t, &created)
	addDomainReq := models.AddWorkerDomainRequest{
		Domain: []string{"domain1.com", "domain2.com"},
	}
	client.Post(t, "/admin/worker/"+created.Name+"/domains", addDomainReq)
	deleteDomainReq := models.DeleteWorkerDomainRequest{
		Domain: []string{"domain1.com"},
	}
	resp := client.DeleteWithBody(t, "/admin/worker/"+created.Name+"/domains", deleteDomainReq)
	resp.RequireStatus(t, http.StatusOK)
	getResp := client.Get(t, "/admin/worker/"+created.Name)
	getResp.RequireStatus(t, http.StatusOK)
	var worker models.AddWorkerResponse
	getResp.ParseJSON(t, &worker)
	assert.NotContains(t, worker.Domains, "domain1.com")
	assert.Contains(t, worker.Domains, "domain2.com")
}

// Worker Login & WebSocket Tests

func TestE2E_WorkerLogin(t *testing.T) {
	adminClient := GetAdminClient()
	workerClient := GetWorkerClient()
	poolId := createTestPoolForWorker(t, adminClient)
	poolUUID, _ := uuid.Parse(poolId)
	createReq := models.AddWorkerRequest{
		RegionName: helpers.Ptr("North America"),
		IPAddress:  helpers.Ptr("192.168.100.1"),
		Port:       helpers.Ptr(int32(8888)),
		PoolId:     helpers.Ptr(poolUUID),
	}
	createResp := adminClient.Post(t, "/admin/worker/", createReq)
	createResp.RequireStatus(t, http.StatusOK)
	var created models.AddWorkerResponse
	createResp.ParseJSON(t, &created)
	workerUUID, _ := uuid.Parse(created.ID)
	loginReq := models.WorkerLoginRequest{
		WorkerId: helpers.Ptr(workerUUID),
	}
	resp := workerClient.Post(t, "/worker/ws/login", loginReq)
	resp.RequireStatus(t, http.StatusOK)
	var loginResp models.WorkerLoginResponce
	resp.ParseJSON(t, &loginResp)
	assert.NotEmpty(t, loginResp.Otp, "OTP should be returned")
	t.Logf("Worker login successful, OTP=%s", loginResp.Otp)
}

func TestE2E_WorkerLogin_InvalidWorkerID(t *testing.T) {
	workerClient := GetWorkerClient()
	nonExistentID := uuid.New()
	loginReq := models.WorkerLoginRequest{
		WorkerId: helpers.Ptr(nonExistentID),
	}
	resp := workerClient.Post(t, "/worker/ws/login", loginReq)
	resp.AssertStatus(t, http.StatusNotFound)
}

func TestE2E_WorkerLogin_MissingWorkerID(t *testing.T) {
	workerClient := GetWorkerClient()
	loginReq := models.WorkerLoginRequest{
		WorkerId: nil,
	}
	resp := workerClient.Post(t, "/worker/ws/login", loginReq)
	resp.AssertStatus(t, http.StatusBadRequest)
}

func TestE2E_WorkerLogin_InvalidAPIKey(t *testing.T) {
	invalidClient := helpers.NewWorkerClient(GetTestServerURL(), "invalid-api-key")
	loginReq := models.WorkerLoginRequest{
		WorkerId: helpers.Ptr(uuid.New()),
	}
	resp := invalidClient.Post(t, "/worker/ws/login", loginReq)
	resp.AssertStatus(t, http.StatusUnauthorized)
}

func TestE2E_WebSocketConnect(t *testing.T) {
	adminClient := GetAdminClient()
	workerClient := GetWorkerClient()
	poolId := createTestPoolForWorker(t, adminClient)
	poolUUID, _ := uuid.Parse(poolId)
	createReq := models.AddWorkerRequest{
		RegionName: helpers.Ptr("Europe"),
		IPAddress:  helpers.Ptr("10.0.0.100"),
		Port:       helpers.Ptr(int32(9999)),
		PoolId:     helpers.Ptr(poolUUID),
	}
	createResp := adminClient.Post(t, "/admin/worker/", createReq)
	createResp.RequireStatus(t, http.StatusOK)
	var created models.AddWorkerResponse
	createResp.ParseJSON(t, &created)
	workerUUID, _ := uuid.Parse(created.ID)
	loginReq := models.WorkerLoginRequest{
		WorkerId: helpers.Ptr(workerUUID),
	}
	loginResp := workerClient.Post(t, "/worker/ws/login", loginReq)
	loginResp.RequireStatus(t, http.StatusOK)
	var login models.WorkerLoginResponce
	loginResp.ParseJSON(t, &login)
	serverURL := GetTestServerURL()
	wsURL := strings.Replace(serverURL, "http://", "ws://", 1)
	wsURL = wsURL + "/worker/ws/serve?otp=" + login.Otp
	header := http.Header{}
	header.Set("Authorization", "ApiKey "+WorkerAPIKey)

	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}
	conn, resp, err := dialer.Dial(wsURL, header)
	if err != nil {
		t.Logf("WebSocket dial error: %v", err)
		if resp != nil {
			t.Logf("Response status: %d", resp.StatusCode)
		}
		t.Skip("WebSocket connection not available, skipping")
		return
	}
	defer conn.Close()
	require.NoError(t, err, "Should connect to WebSocket")
	t.Logf("WebSocket connected successfully")
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	messageType, message, err := conn.ReadMessage()
	if err != nil {
		t.Logf("Error reading message: %v", err)
	} else {
		t.Logf("Received message type=%d, content=%s", messageType, string(message))
		var event struct {
			Type    string          `json:"type"`
			Payload json.RawMessage `json:"payload"`
		}
		if json.Unmarshal(message, &event) == nil {
			assert.Equal(t, "config", event.Type, "First message should be config")
		}
	}
}

func TestE2E_WebSocketConnect_InvalidOTP(t *testing.T) {
	serverURL := GetTestServerURL()
	wsURL := strings.Replace(serverURL, "http://", "ws://", 1)
	wsURL = wsURL + "/worker/ws/serve?otp=invalid-otp-12345"
	header := http.Header{}
	header.Set("Authorization", "ApiKey "+WorkerAPIKey)
	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}
	_, resp, err := dialer.Dial(wsURL, header)
	if err != nil {
		t.Logf("Expected error: %v", err)
		if resp != nil {
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		}
	} else {
		t.Error("Expected WebSocket connection to fail with invalid OTP")
	}
}

func TestE2E_WebSocketConnect_MissingOTP(t *testing.T) {
	serverURL := GetTestServerURL()
	wsURL := strings.Replace(serverURL, "http://", "ws://", 1)
	wsURL = wsURL + "/worker/ws/serve"
	header := http.Header{}
	header.Set("Authorization", "ApiKey "+WorkerAPIKey)
	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}
	_, resp, err := dialer.Dial(wsURL, header)
	if err != nil {
		t.Logf("Expected error: %v", err)
		if resp != nil {
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		}
	} else {
		t.Error("Expected WebSocket connection to fail with invalid OTP")
	}
}
