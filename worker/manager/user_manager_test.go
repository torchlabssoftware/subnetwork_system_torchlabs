package manager

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestUserManager_NewUserManager(t *testing.T) {
	um := NewUserManager()
	if um == nil {
		t.Error("UserManager should not be nil")
	}
	if um.cachedUsers == nil {
		t.Error("cachedUsers should not be nil")
	}
	if um.TTL != time.Hour {
		t.Errorf("TTL should be 1 hour, got %v", um.TTL)
	}
}

func TestUserManager_SetUser(t *testing.T) {
	um := NewUserManager()
	user := createTestUser("testuser", "testpass")
	um.SetUser(user)
	retrievedUser, exists := um.GetUser("testuser")
	if !exists {
		t.Error("User should exist in cache")
	}
	if retrievedUser.Username != user.Username {
		t.Errorf("Username should match, expected %s, got %s", user.Username, retrievedUser.Username)
	}
	if retrievedUser.Password != user.Password {
		t.Errorf("Password should match, expected %s, got %s", user.Password, retrievedUser.Password)
	}
}

func TestUserManager_GetUser_CacheHit(t *testing.T) {
	um := NewUserManager()
	user := createTestUser("testuser", "testpass")
	um.SetUser(user)
	retrievedUser, exists := um.GetUser("testuser")
	if !exists {
		t.Error("User should exist in cache")
	}
	if retrievedUser.ID != user.ID {
		t.Error("User ID should match")
	}
}

func TestUserManager_GetUser_CacheMiss(t *testing.T) {
	um := NewUserManager()

	_, exists := um.GetUser("nonexistent")
	if exists {
		t.Error("Non-existent user should not be found")
	}
}

func TestUserManager_GetUser_Expired(t *testing.T) {
	um := NewUserManager()
	um.TTL = 10 * time.Millisecond
	user := createTestUser("testuser", "testpass")
	um.SetUser(user)
	time.Sleep(50 * time.Millisecond)
	_, exists := um.GetUser("testuser")
	if exists {
		t.Error("Expired user should not be found")
	}
}

func TestUserManager_RemoveUser(t *testing.T) {
	um := NewUserManager()
	user := createTestUser("testuser", "testpass")
	um.SetUser(user)
	um.RemoveUser("testuser")
	_, exists := um.GetUser("testuser")
	if exists {
		t.Error("Removed user should not exist")
	}
}

func TestUserManager_VerifyUser_Valid(t *testing.T) {
	um := NewUserManager()
	user := createTestUser("testuser", "testpass")
	um.SetUser(user)
	onVerifyUser := func(event Event) {
	}
	result := um.VerifyUser("testuser", "testpass", onVerifyUser, "test-pool")
	if !result {
		t.Error("Valid credentials should return true")
	}
}

func TestUserManager_VerifyUser_InvalidPassword(t *testing.T) {
	um := NewUserManager()
	user := createTestUser("testuser", "testpass")
	um.SetUser(user)
	onVerifyUser := func(event Event) {
	}
	result := um.VerifyUser("testuser", "wrongpass", onVerifyUser, "test-pool")
	if result {
		t.Error("Invalid password should return false")
	}
}

func TestUserManager_VerifyUser_InactiveUser(t *testing.T) {
	um := NewUserManager()
	user := createTestUser("testuser", "testpass")
	user.Status = "inactive"
	um.SetUser(user)
	onVerifyUser := func(event Event) {
	}
	result := um.VerifyUser("testuser", "testpass", onVerifyUser, "test-pool")
	if result {
		t.Error("Inactive user should return false")
	}
}

func TestUserManager_VerifyUser_ExceededDataLimit(t *testing.T) {
	um := NewUserManager()
	user := createTestUser("testuser", "testpass")
	user.Pools[0].DataUsage = 2000000
	um.SetUser(user)
	onVerifyUser := func(event Event) {
	}
	result := um.VerifyUser("testuser", "testpass", onVerifyUser, "test-pool")
	if result {
		t.Error("User with exceeded data limit should return false")
	}
}

func TestUserManager_VerifyUser_CaptainCallback(t *testing.T) {
	um := NewUserManager()
	var receivedEvent Event
	onVerifyUser := func(event Event) {
		receivedEvent = event
		go func() {
			time.Sleep(10 * time.Millisecond)
			userPayload := UserPayload{
				ID:          uuid.New(),
				Username:    "testuser",
				Password:    "testpass",
				Status:      "active",
				IpWhitelist: []string{"127.0.0.1"},
				Pools:       []string{"test-pool:1000000:0"},
			}
			um.processVerifyUserResponse(userPayload)
		}()
	}
	result := um.VerifyUser("testuser", "testpass", onVerifyUser, "test-pool")
	if !result {
		t.Error("Captain callback should return true for valid user")
	}
	if receivedEvent.Type != "verify_user" {
		t.Errorf("Should send verify_user event, got %s", receivedEvent.Type)
	}
}

func TestUserManager_VerifyUser_Timeout(t *testing.T) {
	um := NewUserManager()
	onVerifyUser := func(event Event) {
	}
	start := time.Now()
	result := um.VerifyUser("testuser", "testpass", onVerifyUser, "test-pool")
	elapsed := time.Since(start)
	if result {
		t.Error("Timeout should return false")
	}
	if elapsed < 5*time.Second {
		t.Errorf("Should wait at least 5 seconds for timeout, got %v", elapsed)
	}
}

func TestUserManager_ProcessVerifyUserResponse(t *testing.T) {
	um := NewUserManager()
	user := createTestUser("testuser", "testpass")
	userPayload := UserPayload{
		ID:          user.ID,
		Username:    user.Username,
		Password:    user.Password,
		Status:      user.Status,
		IpWhitelist: user.IpWhitelist,
		Pools:       []string{"test-pool:1000000:0"},
	}
	onVerifyUser := func(event Event) {
		go func() {
			um.processVerifyUserResponse(userPayload)
		}()

	}
	um.VerifyUser(user.Username, user.Password, onVerifyUser, "test-pool")
	user, exists := um.GetUser("testuser")
	if !exists {
		t.Error("User should be cached after processing response")
	}
	if userPayload.Username != user.Username {
		t.Error("Username should match")
	}
	if userPayload.Status != user.Status {
		t.Error("Status should match")
	}
	if len(user.Pools) != 1 {
		t.Errorf("Should have one pool, got %d", len(user.Pools))
	}
	if user.Pools[0].Tag != "test-pool" {
		t.Errorf("Pool tag should match, expected test-pool, got %s", user.Pools[0].Tag)
	}
	if user.Pools[0].DataLimit != 1000000 {
		t.Errorf("Data limit should match, expected 1000000, got %d", user.Pools[0].DataLimit)
	}
}

func TestUserManager_AddConnection(t *testing.T) {
	um := NewUserManager()
	user := createTestUser("testuser", "testpass")
	um.SetUser(user)
	for i := 0; i < 50; i++ {
		err := um.addConnection("testuser")
		if err != nil {
			t.Errorf("Should be able to add connection within limit: %v", err)
		}
	}
	err := um.addConnection("testuser")
	if err == nil {
		t.Error("Should fail when exceeding connection limit")
	}
}

func TestUserManager_AddConnection_UserNotFound(t *testing.T) {
	um := NewUserManager()
	err := um.addConnection("nonexistent")
	if err == nil {
		t.Error("Should fail for non-existent user")
	}
}

func TestUserManager_RemoveConnection(t *testing.T) {
	um := NewUserManager()
	user := createTestUser("testuser", "testpass")
	um.SetUser(user)
	um.addConnection("testuser")
	um.removeConnection("testuser")
}

func TestUserManager_CleanupLoop(t *testing.T) {
	um := NewUserManager()
	um.TTL = 1 * time.Second
	go func() {
		um.cleanupLoop(2 * time.Second)
	}()
	user := createTestUser("testuser", "testpass")
	um.SetUser(user)
	_, exists := um.GetUser("testuser")
	if !exists {
		t.Error("User should exist before cleanup")
	}
	time.Sleep(5 * time.Second)
	_, exists = um.GetUser("testuser")
	if exists {
		t.Error("Expired user should be cleaned up")
	}
}

func createTestUser(username, password string) *User {
	userID := uuid.New()
	user := &User{
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
	return user
}
