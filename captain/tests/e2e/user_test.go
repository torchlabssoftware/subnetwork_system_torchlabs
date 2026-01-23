package e2e

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	models "github.com/torchlabssoftware/subnetwork_system/internal/server/models"
	"github.com/torchlabssoftware/subnetwork_system/tests/e2e/helpers"
)

func TestE2E_CreateUser(t *testing.T) {
	client := GetAdminClient()
	reqBody := models.CreateUserRequest{
		IpWhiteList: helpers.Ptr([]string{"192.168.1.1", "10.0.0.0/24"}),
		AllowPools:  helpers.Ptr([]models.PoolDataStat{}),
	}
	resp := client.Post(t, "/admin/users/", reqBody)
	resp.RequireStatus(t, http.StatusCreated)
	var result models.CreateUserResponce
	resp.ParseJSON(t, &result)
	assert.NotEqual(t, uuid.Nil, result.Id, "User ID should not be nil")
	assert.NotEmpty(t, result.Username, "Username should be generated")
	assert.NotEmpty(t, result.Password, "Password should be generated")
	assert.Equal(t, "active", result.Status, "Default status should be active")
	assert.NotEmpty(t, result.Created_at, "Created_at should be set")
	t.Logf("Created user: ID=%s, Username=%s", result.Id, result.Username)
}

func TestE2E_GetUsers(t *testing.T) {
	client := GetAdminClient()
	createReq := models.CreateUserRequest{}
	createResp := client.Post(t, "/admin/users/", createReq)
	createResp.RequireStatus(t, http.StatusCreated)
	resp := client.Get(t, "/admin/users/")
	resp.RequireStatus(t, http.StatusOK)
	var users []models.GetUserByIdResponce
	resp.ParseJSON(t, &users)
	assert.GreaterOrEqual(t, len(users), 1, "Should have at least one user")
	t.Logf("Found %d users", len(users))
}

func TestE2E_GetUserById(t *testing.T) {
	client := GetAdminClient()
	createReq := models.CreateUserRequest{
		IpWhiteList: helpers.Ptr([]string{"172.16.0.1"}),
	}
	createResp := client.Post(t, "/admin/users/", createReq)
	createResp.RequireStatus(t, http.StatusCreated)
	var created models.CreateUserResponce
	createResp.ParseJSON(t, &created)
	resp := client.Get(t, "/admin/users/"+created.Id.String())
	resp.RequireStatus(t, http.StatusOK)
	var user models.GetUserByIdResponce
	resp.ParseJSON(t, &user)
	assert.Equal(t, created.Id, user.Id)
	assert.Equal(t, created.Username, user.Username)
	assert.Equal(t, "active", user.Status)
}

func TestE2E_GetUserById_NotFound(t *testing.T) {
	client := GetAdminClient()
	nonExistentID := uuid.New().String()
	resp := client.Get(t, "/admin/users/"+nonExistentID)
	resp.AssertStatus(t, http.StatusNotFound)
}

func TestE2E_UpdateUserStatus(t *testing.T) {
	client := GetAdminClient()
	createReq := models.CreateUserRequest{}
	createResp := client.Post(t, "/admin/users/", createReq)
	createResp.RequireStatus(t, http.StatusCreated)
	var created models.CreateUserResponce
	createResp.ParseJSON(t, &created)
	updateReq := models.UpdateUserRequest{
		Status: helpers.Ptr("suspended"),
	}
	resp := client.DoRequest(t, helpers.RequestOptions{
		Method: http.MethodPatch,
		Path:   "/admin/users/" + created.Id.String(),
		Body:   updateReq,
	})
	resp.RequireStatus(t, http.StatusOK)
	var updated models.UpdateUserResponce
	resp.ParseJSON(t, &updated)
	assert.Equal(t, created.Id, updated.Id)
	assert.Equal(t, "suspended", updated.Status)
	getResp := client.Get(t, "/admin/users/"+created.Id.String())
	getResp.RequireStatus(t, http.StatusOK)
	var user models.GetUserByIdResponce
	getResp.ParseJSON(t, &user)
	assert.Equal(t, "suspended", user.Status)
}

func TestE2E_DeleteUser(t *testing.T) {
	client := GetAdminClient()
	createReq := models.CreateUserRequest{}
	createResp := client.Post(t, "/admin/users/", createReq)
	createResp.RequireStatus(t, http.StatusCreated)
	var created models.CreateUserResponce
	createResp.ParseJSON(t, &created)
	resp := client.Delete(t, "/admin/users/"+created.Id.String())
	resp.RequireStatus(t, http.StatusOK)
	getResp := client.Get(t, "/admin/users/"+created.Id.String())
	getResp.AssertStatus(t, http.StatusNotFound)
}

func TestE2E_UserIpWhitelist(t *testing.T) {
	client := GetAdminClient()
	createReq := models.CreateUserRequest{}
	createResp := client.Post(t, "/admin/users/", createReq)
	createResp.RequireStatus(t, http.StatusCreated)
	var created models.CreateUserResponce
	createResp.ParseJSON(t, &created)
	userID := created.Id.String()
	addReq := models.AddUserIpwhitelistRequest{
		IpWhitelist: []string{"192.168.1.100", "10.0.0.0/8"},
	}
	addResp := client.Post(t, "/admin/users/"+userID+"/ipwhitelist", addReq)
	addResp.RequireStatus(t, http.StatusCreated)
	getResp := client.Get(t, "/admin/users/"+userID+"/ipwhitelist")
	getResp.RequireStatus(t, http.StatusOK)
	var whitelist models.GetUserIpwhitelistResponce
	getResp.ParseJSON(t, &whitelist)
	assert.Contains(t, whitelist.IpWhitelist, "192.168.1.100")
	assert.Contains(t, whitelist.IpWhitelist, "10.0.0.0/8")
	removeReq := models.DeleteUserIpwhitelistRequest{
		IpCidr: []string{"192.168.1.100"},
	}
	removeResp := client.DeleteWithBody(t, "/admin/users/"+userID+"/ipwhitelist", removeReq)
	removeResp.RequireStatus(t, http.StatusOK)
	verifyResp := client.Get(t, "/admin/users/"+userID+"/ipwhitelist")
	verifyResp.RequireStatus(t, http.StatusOK)
	var updatedWhitelist models.GetUserIpwhitelistResponce
	verifyResp.ParseJSON(t, &updatedWhitelist)
	assert.NotContains(t, updatedWhitelist.IpWhitelist, "192.168.1.100")
}

func TestE2E_UserAuthentication_InvalidAPIKey(t *testing.T) {
	client := helpers.NewAdminClient(GetTestServerURL(), "invalid-api-key")
	resp := client.Get(t, "/admin/users/")
	resp.AssertStatus(t, http.StatusUnauthorized)
}

func TestE2E_UserPools(t *testing.T) {
	client := GetAdminClient()
	createReq := models.CreateUserRequest{}
	createResp := client.Post(t, "/admin/users/", createReq)
	createResp.RequireStatus(t, http.StatusCreated)
	var created models.CreateUserResponce
	createResp.ParseJSON(t, &created)
	userID := created.Id.String()
	addReq := models.AddUserPoolRequest{
		UserPool: []models.PoolDataStat{
			{
				Pool:      "netnutusa",
				DataLimit: 1000000,
			},
			{
				Pool:      "netnuteu",
				DataLimit: 2000000,
			},
		},
	}
	addResp := client.Post(t, "/admin/users/"+userID+"/pools", addReq)
	addResp.RequireStatus(t, http.StatusCreated)
	getResp := client.Get(t, "/admin/users/"+userID+"/pools")
	getResp.RequireStatus(t, http.StatusOK)
	var pools models.GetUserPoolResponce
	getResp.ParseJSON(t, &pools)
	assert.Contains(t, pools.Pools, "netnutusa")
	assert.Contains(t, pools.Pools, "netnuteu")
	removeReq := models.DeleteUserpoolRequest{
		UserPool: []string{"netnutusa"},
	}
	removeResp := client.DeleteWithBody(t, "/admin/users/"+userID+"/pools", removeReq)
	removeResp.RequireStatus(t, http.StatusOK)
	verifyResp := client.Get(t, "/admin/users/"+userID+"/pools")
	verifyResp.RequireStatus(t, http.StatusOK)
	var updatedPools models.GetUserPoolResponce
	verifyResp.ParseJSON(t, &updatedPools)
	assert.NotContains(t, updatedPools.Pools, "netnutusa")
	assert.Contains(t, updatedPools.Pools, "netnuteu")
}

func TestE2E_UserDataUsage(t *testing.T) {
	client := GetAdminClient()
	createReq := models.CreateUserRequest{}
	createResp := client.Post(t, "/admin/users/", createReq)
	createResp.RequireStatus(t, http.StatusCreated)
	var created models.CreateUserResponce
	createResp.ParseJSON(t, &created)
	userID := created.Id.String()
	addPoolReq := models.AddUserPoolRequest{
		UserPool: []models.PoolDataStat{
			{
				Pool:      "netnutusa",
				DataLimit: 5000000,
			},
		},
	}
	addPoolResp := client.Post(t, "/admin/users/"+userID+"/pools", addPoolReq)
	addPoolResp.RequireStatus(t, http.StatusCreated)
	resp := client.Get(t, "/admin/users/"+userID+"/data-usage")
	resp.RequireStatus(t, http.StatusOK)
	var dataUsage []models.GetDatausageReponce
	resp.ParseJSON(t, &dataUsage)
	assert.GreaterOrEqual(t, len(dataUsage), 1, "Should have at least one data usage record")
	found := false
	for _, du := range dataUsage {
		if du.PoolTag == "netnutusa" {
			assert.Equal(t, int64(5000000), du.DataLimit)
			assert.GreaterOrEqual(t, du.DataUsage, int64(0))
			found = true
			break
		}
	}
	assert.True(t, found, "Should find data usage for test pool")
}
