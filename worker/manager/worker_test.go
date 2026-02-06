package manager

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestWorkerManager_NewWorkerManager(t *testing.T) {
	workerID := uuid.New().String()
	baseURL := "https://test-captain.com"
	apiKey := "test-api-key"
	wm, err := NewWorkerManager(workerID, baseURL, apiKey)
	if err != nil {
		t.Errorf("Should not return error: %v", err)
	}
	if wm == nil {
		t.Error("WorkerManager should not be nil")
	}
	if wm.Worker.ID.String() != workerID {
		t.Errorf("Worker ID should be set, expected %s, got %s", workerID, wm.Worker.ID.String())
	}
	if wm.Worker.CaptainURL != baseURL {
		t.Errorf("Captain URL should be set, expected %s, got %s", baseURL, wm.Worker.CaptainURL)
	}
	if wm.Worker.APIKey != apiKey {
		t.Errorf("API key should be set, expected %s, got %s", apiKey, wm.Worker.APIKey)
	}
	if wm.upstreamManager == nil {
		t.Error("UpstreamManager should be initialized")
	}
	if wm.HealthCollector == nil {
		t.Error("HealthCollector should be initialized")
	}
	if wm.userManager == nil {
		t.Error("UserManager should be initialized")
	}
}

func TestWorkerManager_NewWorkerManager_InvalidID(t *testing.T) {
	invalidID := "invalid-uuid"
	baseURL := "https://test-captain.com"
	apiKey := "test-api-key"
	wm, err := NewWorkerManager(invalidID, baseURL, apiKey)
	if err == nil {
		t.Error("Should return error for invalid UUID")
	}
	if wm != nil {
		t.Error("WorkerManager should be nil for invalid UUID")
	}
}

func TestWorkerManager_NewWorkerManager_EmptyID(t *testing.T) {
	emptyID := ""
	baseURL := "https://test-captain.com"
	apiKey := "test-api-key"
	wm, err := NewWorkerManager(emptyID, baseURL, apiKey)
	if err == nil {
		t.Error("Should return error for empty UUID")
	}
	if wm != nil {
		t.Error("WorkerManager should be nil for empty UUID")
	}
}

func TestWorkerManager_Start(t *testing.T) {
	workerID := uuid.New().String()
	baseURL := "https://test-captain.com"
	apiKey := "test-api-key"

	wm, err := NewWorkerManager(workerID, baseURL, apiKey)
	if err != nil {
		t.Fatalf("Failed to create WorkerManager: %v", err)
	}
	go wm.Start()
	time.Sleep(100 * time.Millisecond)
}

func TestWorkerManager_Login(t *testing.T) {
	workerID := uuid.New().String()
	baseURL := "https://test-captain.com"
	apiKey := "test-api-key"
	wm, err := NewWorkerManager(workerID, baseURL, apiKey)
	if err != nil {
		t.Fatalf("Failed to create WorkerManager: %v", err)
	}
	otp, err := wm.login()
	if err == nil {
		t.Error("Should return error when Captain is not available")
	}
	if otp != "" {
		t.Error("OTP should be empty when login fails")
	}
}

func TestWorkerManager_ProcessConfig(t *testing.T) {
	workerID := uuid.New().String()
	baseURL := "https://test-captain.com"
	apiKey := "test-api-key"
	wm, err := NewWorkerManager(workerID, baseURL, apiKey)
	if err != nil {
		t.Fatalf("Failed to create WorkerManager: %v", err)
	}
	config := createTestConfigPayloadForWorker()
	wm.processConfig(config)
	if wm.Worker.Name != config.WorkerName {
		t.Errorf("Worker name should be updated, expected %s, got %s", config.WorkerName, wm.Worker.Name)
	}
	if wm.Worker.Region != config.Region {
		t.Errorf("Worker region should be updated, expected %s, got %s", config.Region, wm.Worker.Region)
	}
	if wm.Worker.Pool == nil {
		t.Error("Pool should be created")
	}
	if wm.Worker.Pool.PoolTag != config.PoolTag {
		t.Errorf("Pool tag should match, expected %s, got %s", config.PoolTag, wm.Worker.Pool.PoolTag)
	}
	if wm.Worker.Pool.PoolPort != config.PoolPort {
		t.Errorf("Pool port should match, expected %d, got %d", config.PoolPort, wm.Worker.Pool.PoolPort)
	}
	if wm.Worker.Pool.PoolSubdomain != config.PoolSubdomain {
		t.Errorf("Pool subdomain should match, expected %s, got %s", config.PoolSubdomain, wm.Worker.Pool.PoolSubdomain)
	}
}

func TestWorkerManager_ProcessConfig_Upstreams(t *testing.T) {
	workerID := uuid.New().String()
	baseURL := "https://test-captain.com"
	apiKey := "test-api-key"
	wm, err := NewWorkerManager(workerID, baseURL, apiKey)
	if err != nil {
		t.Fatalf("Failed to create WorkerManager: %v", err)
	}
	config := ConfigPayload{
		WorkerName:    "test-worker",
		Region:        "us-east-1",
		PoolID:        uuid.New(),
		PoolTag:       "test-pool",
		PoolPort:      8080,
		PoolSubdomain: "test",
		Upstreams: []UpstreamConfig{
			{
				UpstreamID:       uuid.New(),
				UpstreamTag:      "upstream1",
				UpstreamFormat:   "http",
				UpstreamUsername: "user1",
				UpstreamPassword: "pass1",
				UpstreamHost:     "127.0.0.1",
				UpstreamPort:     3128,
				UpstreamProvider: "provider1",
				Weight:           1,
			},
			{
				UpstreamID:       uuid.New(),
				UpstreamTag:      "upstream2",
				UpstreamFormat:   "socks5",
				UpstreamUsername: "user2",
				UpstreamPassword: "pass2",
				UpstreamHost:     "127.0.0.2",
				UpstreamPort:     1080,
				UpstreamProvider: "provider2",
				Weight:           2,
			},
		},
	}
	wm.processConfig(config)
	if !wm.upstreamManager.HasUpstreams() {
		t.Error("Should have upstreams after processing config")
	}
	upstream1 := wm.upstreamManager.Next()
	upstream2 := wm.upstreamManager.Next()
	if upstream1 == nil || upstream2 == nil {
		t.Error("Should be able to select upstreams")
	}
	if upstream1.UpstreamTag == upstream2.UpstreamTag {
		t.Error("Should select different upstreams in round-robin")
	}
}

func TestWorkerManager_ProcessUserChange(t *testing.T) {
	workerID := uuid.New().String()
	baseURL := "https://test-captain.com"
	apiKey := "test-api-key"

	wm, err := NewWorkerManager(workerID, baseURL, apiKey)
	if err != nil {
		t.Fatalf("Failed to create WorkerManager: %v", err)
	}
	user := createTestUserForWorker("testuser", "testpass")
	wm.userManager.SetUser(user)
	wm.processUserChange("testuser")
	_, exists := wm.userManager.GetUser("testuser")
	if exists {
		t.Error("User should be removed after user change event")
	}
}

func TestWorkerManager_HasUpstreams(t *testing.T) {
	workerID := uuid.New().String()
	baseURL := "https://test-captain.com"
	apiKey := "test-api-key"
	wm, err := NewWorkerManager(workerID, baseURL, apiKey)
	if err != nil {
		t.Fatalf("Failed to create WorkerManager: %v", err)
	}
	if wm.HasUpstreams() {
		t.Error("Should not have upstreams initially")
	}
	config := createTestConfigPayload()
	wm.processConfig(config)
	if !wm.HasUpstreams() {
		t.Error("Should have upstreams after processing config")
	}
}

func TestWorkerManager_AddUserConnection(t *testing.T) {
	workerID := uuid.New().String()
	baseURL := "https://test-captain.com"
	apiKey := "test-api-key"

	wm, err := NewWorkerManager(workerID, baseURL, apiKey)
	if err != nil {
		t.Fatalf("Failed to create WorkerManager: %v", err)
	}
	user := createTestUserForWorker("testuser", "testpass")
	wm.userManager.SetUser(user)
	err = wm.AddUserConnection("testuser")
	if err != nil {
		t.Errorf("Should be able to add connection: %v", err)
	}
	err = wm.AddUserConnection("invaliduser")
	if err == nil {
		t.Error("Should not be able to add connection for invalid user")
	}
}

func TestWorkerManager_SendHealthTelemetry(t *testing.T) {
	workerID := uuid.New().String()
	baseURL := "https://test-captain.com"
	apiKey := "test-api-key"
	wm, err := NewWorkerManager(workerID, baseURL, apiKey)
	if err != nil {
		t.Fatalf("Failed to create WorkerManager: %v", err)
	}
	wm.SendHealthTelemetry()
}

func createTestUserForWorker(username, password string) *User {
	userID := uuid.New()
	return &User{
		ID:          userID,
		Username:    username,
		Password:    password,
		Status:      "active",
		IpWhitelist: []string{"127.0.0.1"},
		Pools: []PoolLimit{
			{
				Tag:       "test-pool",
				DataLimit: 1000000,
				DataUsage: 0,
			},
		},
		Sessions: make(map[string]Upstream),
	}
}

func createTestConfigPayloadForWorker() ConfigPayload {
	return ConfigPayload{
		WorkerName:    "test-worker",
		Region:        "us-east-1",
		PoolID:        uuid.New(),
		PoolTag:       "test-pool",
		PoolPort:      8080,
		PoolSubdomain: "test",
		Upstreams: []UpstreamConfig{
			{
				UpstreamID:       uuid.New(),
				UpstreamTag:      "test-upstream",
				UpstreamFormat:   "http",
				UpstreamUsername: "user",
				UpstreamPassword: "pass",
				UpstreamHost:     "127.0.0.1",
				UpstreamPort:     3128,
				UpstreamProvider: "test",
				Weight:           1,
			},
		},
	}
}
