package e2e

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestE2E_DatabaseConnection(t *testing.T) {
	db := GetTestDB()
	if db == nil {
		t.Fatal("Database connection not initialized")
	}
	err := db.Ping()
	assert.NoError(t, err, "Database should be reachable")
}

func TestE2E_AdminAuthentication(t *testing.T) {
	client := GetAdminClient()
	resp := client.Get(t, "/admin/users/")
	assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode,
		"Admin API key should be valid. Got: %s", resp.String())
}

func TestE2E_WorkerAuthentication(t *testing.T) {
	client := GetWorkerClient()
	resp := client.Post(t, "/worker/ws/login", map[string]interface{}{})
	assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode,
		"Worker API key should be valid. Got: %s", resp.String())
}
