package e2e

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	models "github.com/torchlabssoftware/subnetwork_system/internal/server/models"
	"github.com/torchlabssoftware/subnetwork_system/tests/e2e/helpers"
)

// Region Tests

func TestE2E_CreateRegion(t *testing.T) {
	client := GetAdminClient()
	regionName := "Test Region " + uuid.New().String()[:8]
	reqBody := models.CreateRegionRequest{
		Name: helpers.Ptr(regionName),
	}
	resp := client.Post(t, "/admin/pools/region", reqBody)
	resp.RequireStatus(t, http.StatusCreated)
	var result models.CreateRegionResponce
	resp.ParseJSON(t, &result)
	assert.NotEqual(t, uuid.Nil, result.Id)
	assert.Equal(t, regionName, result.Name)
	assert.NotEmpty(t, result.CreatedAt)
	t.Logf("Created region: ID=%s, Name=%s", result.Id, result.Name)
}

func TestE2E_GetRegions(t *testing.T) {
	client := GetAdminClient()
	regionName := "Test Region " + uuid.New().String()[:8]
	createReq := models.CreateRegionRequest{
		Name: helpers.Ptr(regionName),
	}
	client.Post(t, "/admin/pools/region", createReq)
	resp := client.Get(t, "/admin/pools/region")
	resp.RequireStatus(t, http.StatusOK)
	var regions []models.GetRegionResponce
	resp.ParseJSON(t, &regions)
	assert.GreaterOrEqual(t, len(regions), 1, "Should have at least one region")
	t.Logf("Found %d regions", len(regions))
}

func TestE2E_DeleteRegion(t *testing.T) {
	client := GetAdminClient()
	regionName := "DeleteMe " + uuid.New().String()[:8]
	createReq := models.CreateRegionRequest{
		Name: helpers.Ptr(regionName),
	}
	createResp := client.Post(t, "/admin/pools/region", createReq)
	createResp.RequireStatus(t, http.StatusCreated)
	deleteREq := models.DeleteRegionRequest{
		Name: createReq.Name,
	}
	resp := client.DoRequest(t, helpers.RequestOptions{
		Method: http.MethodDelete,
		Path:   "/admin/pools/region",
		Body:   deleteREq,
	})
	resp.RequireStatus(t, http.StatusOK)
}

// Country Tests

func TestE2E_CreateCountry(t *testing.T) {
	client := GetAdminClient()
	regionName := "Country Test Region " + uuid.New().String()[:8]
	regionReq := models.CreateRegionRequest{
		Name: helpers.Ptr(regionName),
	}
	regionResp := client.Post(t, "/admin/pools/region", regionReq)
	regionResp.RequireStatus(t, http.StatusCreated)
	var region models.CreateRegionResponce
	regionResp.ParseJSON(t, &region)
	countryCode := "T" + uuid.New().String()[:2]
	reqBody := models.CreateCountryRequest{
		Name:     helpers.Ptr("Test Country"),
		Code:     helpers.Ptr(countryCode),
		RegionId: helpers.Ptr(region.Id),
	}
	resp := client.Post(t, "/admin/pools/country", reqBody)
	resp.RequireStatus(t, http.StatusCreated)
	var result models.CreateCountryResponce
	resp.ParseJSON(t, &result)
	assert.NotEqual(t, uuid.Nil, result.Id)
	assert.Equal(t, "Test Country", result.Name)
	assert.Equal(t, countryCode, result.Code)
	assert.Equal(t, region.Id, result.RegionId)
	t.Logf("Created country: ID=%s, Code=%s", result.Id, result.Code)
}

func TestE2E_GetCountries(t *testing.T) {
	client := GetAdminClient()
	resp := client.Get(t, "/admin/pools/country")
	resp.RequireStatus(t, http.StatusOK)
	var countries []models.GetCountryResponce
	resp.ParseJSON(t, &countries)
	t.Logf("Found %d countries", len(countries))
}

// Upstream Tests

func TestE2E_CreateUpstream(t *testing.T) {
	client := GetAdminClient()
	upstreamTag := "upstream-" + uuid.New().String()[:8]
	reqBody := models.CreateUpstreamRequest{
		Tag:              helpers.Ptr(upstreamTag),
		UpstreamProvider: helpers.Ptr("test-provider"),
		ConfigFormat:     helpers.Ptr("user:pass@host:port"),
		Username:         helpers.Ptr("testuser"),
		Password:         helpers.Ptr("testpass"),
		Port:             helpers.Ptr(8080),
		Domain:           helpers.Ptr("proxy.test.com"),
	}
	resp := client.Post(t, "/admin/pools/upstream", reqBody)
	resp.RequireStatus(t, http.StatusCreated)
	var result models.CreateUpstreamResponce
	resp.ParseJSON(t, &result)
	assert.NotEqual(t, uuid.Nil, result.Id)
	assert.Equal(t, upstreamTag, result.Tag)
	assert.Equal(t, "test-provider", result.UpstreamProvider)
	assert.Equal(t, 8080, result.Port)
	t.Logf("Created upstream: ID=%s, Tag=%s", result.Id, result.Tag)
}

func TestE2E_GetUpstreams(t *testing.T) {
	client := GetAdminClient()
	upstreamTag := "list-upstream-" + uuid.New().String()[:8]
	createReq := models.CreateUpstreamRequest{
		Tag:              helpers.Ptr(upstreamTag),
		UpstreamProvider: helpers.Ptr("provider"),
		ConfigFormat:     helpers.Ptr("format"),
		Username:         helpers.Ptr("user"),
		Password:         helpers.Ptr("pass"),
		Port:             helpers.Ptr(8080),
		Domain:           helpers.Ptr("proxy.example.com"),
	}
	client.Post(t, "/admin/pools/upstream", createReq)
	resp := client.Get(t, "/admin/pools/upstream")
	resp.RequireStatus(t, http.StatusOK)
	var upstreams []models.GetUpstreamResponce
	resp.ParseJSON(t, &upstreams)
	assert.GreaterOrEqual(t, len(upstreams), 1)
	t.Logf("Found %d upstreams", len(upstreams))
}

func TestE2E_DeleteUpstream(t *testing.T) {
	client := GetAdminClient()
	upstreamTag := "delete-upstream-" + uuid.New().String()[:8]
	createReq := models.CreateUpstreamRequest{
		Tag:              helpers.Ptr(upstreamTag),
		UpstreamProvider: helpers.Ptr("provider"),
		ConfigFormat:     helpers.Ptr("format"),
		Username:         helpers.Ptr("user"),
		Password:         helpers.Ptr("pass"),
		Port:             helpers.Ptr(8080),
		Domain:           helpers.Ptr("proxy.delete.com"),
	}
	createResp := client.Post(t, "/admin/pools/upstream", createReq)
	createResp.RequireStatus(t, http.StatusCreated)
	deleteReq := models.DeleteUpstreamRequest{
		Tag: helpers.Ptr(upstreamTag),
	}
	resp := client.DeleteWithBody(t, "/admin/pools/upstream", deleteReq)
	resp.RequireStatus(t, http.StatusOK)
}

// Pool Tests

func TestE2E_CreatePool(t *testing.T) {
	client := GetAdminClient()
	regionName := "Pool Test Region " + uuid.New().String()[:8]
	regionReq := models.CreateRegionRequest{
		Name: helpers.Ptr(regionName),
	}
	regionResp := client.Post(t, "/admin/pools/region", regionReq)
	regionResp.RequireStatus(t, http.StatusCreated)
	var region models.CreateRegionResponce
	regionResp.ParseJSON(t, &region)
	poolTag := "pool-" + uuid.New().String()[:8]
	reqBody := models.CreatePoolRequest{
		Tag:       helpers.Ptr(poolTag),
		RegionId:  helpers.Ptr(region.Id),
		Subdomain: helpers.Ptr("test-subdomain"),
		Port:      helpers.Ptr(int32(8888)),
		UpStreams: helpers.Ptr([]models.CreateUpstreamWeightRequest{}),
	}
	resp := client.Post(t, "/admin/pools/", reqBody)
	resp.RequireStatus(t, http.StatusCreated)
	var result models.CreatePoolResponce
	resp.ParseJSON(t, &result)
	assert.NotEqual(t, uuid.Nil, result.Id)
	assert.Equal(t, poolTag, result.Tag)
	assert.Equal(t, region.Id, result.RegionId)
	assert.Equal(t, int32(8888), result.Port)
	t.Logf("Created pool: ID=%s, Tag=%s", result.Id, result.Tag)
}

func TestE2E_GetPools(t *testing.T) {
	client := GetAdminClient()
	resp := client.Get(t, "/admin/pools/")
	resp.RequireStatus(t, http.StatusOK)
	var pools []models.GetPoolsResponse
	resp.ParseJSON(t, &pools)
	t.Logf("Found %d pools", len(pools))
}

func TestE2E_GetPoolByTag(t *testing.T) {
	client := GetAdminClient()
	regionName := "Get Pool Region " + uuid.New().String()[:8]
	regionReq := models.CreateRegionRequest{
		Name: helpers.Ptr(regionName),
	}
	regionResp := client.Post(t, "/admin/pools/region", regionReq)
	regionResp.RequireStatus(t, http.StatusCreated)
	var region models.CreateRegionResponce
	regionResp.ParseJSON(t, &region)
	poolTag := "get-pool-" + uuid.New().String()[:8]
	createReq := models.CreatePoolRequest{
		Tag:       helpers.Ptr(poolTag),
		RegionId:  helpers.Ptr(region.Id),
		Subdomain: helpers.Ptr("get-test"),
		Port:      helpers.Ptr(int32(9999)),
	}
	createResp := client.Post(t, "/admin/pools/", createReq)
	createResp.RequireStatus(t, http.StatusCreated)
	resp := client.Get(t, "/admin/pools/"+poolTag)
	resp.RequireStatus(t, http.StatusOK)
	var pool models.GetPoolsResponse
	resp.ParseJSON(t, &pool)
	assert.Equal(t, poolTag, pool.Tag)
}

func TestE2E_UpdatePool(t *testing.T) {
	client := GetAdminClient()
	regionName := "Update Pool Region " + uuid.New().String()[:8]
	regionReq := models.CreateRegionRequest{
		Name: helpers.Ptr(regionName),
	}
	regionResp := client.Post(t, "/admin/pools/region", regionReq)
	regionResp.RequireStatus(t, http.StatusCreated)
	var region models.CreateRegionResponce
	regionResp.ParseJSON(t, &region)
	poolTag := "update-pool-" + uuid.New().String()[:8]
	createReq := models.CreatePoolRequest{
		Tag:       helpers.Ptr(poolTag),
		RegionId:  helpers.Ptr(region.Id),
		Subdomain: helpers.Ptr("original"),
		Port:      helpers.Ptr(int32(7777)),
	}
	createResp := client.Post(t, "/admin/pools/", createReq)
	createResp.RequireStatus(t, http.StatusCreated)
	updateReq := models.UpdatePoolRequest{
		Subdomain: helpers.Ptr("updated-subdomain"),
		Port:      helpers.Ptr(int32(8888)),
	}
	resp := client.Put(t, "/admin/pools/"+poolTag, updateReq)
	resp.RequireStatus(t, http.StatusOK)
	var updated models.CreatePoolResponce
	resp.ParseJSON(t, &updated)
	assert.Equal(t, "updated-subdomain", updated.Subdomain)
	assert.Equal(t, int32(8888), updated.Port)
}

func TestE2E_DeletePool(t *testing.T) {
	client := GetAdminClient()
	regionName := "Delete Pool Region " + uuid.New().String()[:8]
	regionReq := models.CreateRegionRequest{
		Name: helpers.Ptr(regionName),
	}
	regionResp := client.Post(t, "/admin/pools/region", regionReq)
	regionResp.RequireStatus(t, http.StatusCreated)
	var region models.CreateRegionResponce
	regionResp.ParseJSON(t, &region)
	poolTag := "delete-pool-" + uuid.New().String()[:8]
	createReq := models.CreatePoolRequest{
		Tag:       helpers.Ptr(poolTag),
		RegionId:  helpers.Ptr(region.Id),
		Subdomain: helpers.Ptr("to-delete"),
		Port:      helpers.Ptr(int32(6666)),
	}
	createResp := client.Post(t, "/admin/pools/", createReq)
	createResp.RequireStatus(t, http.StatusCreated)
	resp := client.Delete(t, "/admin/pools/"+poolTag)
	resp.RequireStatus(t, http.StatusOK)
	getResp := client.Get(t, "/admin/pools/"+poolTag)
	getResp.AssertStatus(t, http.StatusNotFound)
}

func TestE2E_UpstreamPoolWeightWorkflow(t *testing.T) {
	client := GetAdminClient()
	regionName := "Upstream Pool Region " + uuid.New().String()[:8]
	regionReq := models.CreateRegionRequest{
		Name: helpers.Ptr(regionName),
	}
	regionResp := client.Post(t, "/admin/pools/region", regionReq)
	regionResp.RequireStatus(t, http.StatusCreated)
	var region models.CreateRegionResponce
	regionResp.ParseJSON(t, &region)
	poolTag := "upstream-weight-pool-" + uuid.New().String()[:8]
	poolReq := models.CreatePoolRequest{
		Tag:       helpers.Ptr(poolTag),
		RegionId:  helpers.Ptr(region.Id),
		Subdomain: helpers.Ptr("weight-test"),
		Port:      helpers.Ptr(int32(5555)),
	}
	poolResp := client.Post(t, "/admin/pools/", poolReq)
	poolResp.RequireStatus(t, http.StatusCreated)
	upstreamTag := "weight-upstream-" + uuid.New().String()[:8]
	upstreamReq := models.CreateUpstreamRequest{
		Tag:              helpers.Ptr(upstreamTag),
		UpstreamProvider: helpers.Ptr("provider"),
		ConfigFormat:     helpers.Ptr("format"),
		Username:         helpers.Ptr("user"),
		Password:         helpers.Ptr("pass"),
		Port:             helpers.Ptr(8080),
		Domain:           helpers.Ptr("proxy.weight.com"),
	}
	upstreamResp := client.Post(t, "/admin/pools/upstream", upstreamReq)
	upstreamResp.RequireStatus(t, http.StatusCreated)
	addWeightReq := models.AddPoolUpstreamWeightRequest{
		PoolTag:     helpers.Ptr(poolTag),
		UpstreamTag: helpers.Ptr(upstreamTag),
		Weight:      helpers.Ptr(int32(10)),
	}
	resp := client.Post(t, "/admin/pools/weight", addWeightReq)
	resp.RequireStatus(t, http.StatusCreated)
	t.Logf("Added upstream %s to pool %s with weight", upstreamTag, poolTag)
	getPoolResp := client.DoRequest(t, helpers.RequestOptions{
		Method: http.MethodDelete,
		Path:   "/admin/pools/weight",
		Body: models.DeletePoolUpstreamWeightRequest{
			PoolTag:     helpers.Ptr(poolTag),
			UpstreamTag: helpers.Ptr(upstreamTag),
		},
	})
	getPoolResp.RequireStatus(t, http.StatusOK)
	t.Logf("Removed upstream %s from pool %s", upstreamTag, poolTag)
	//assert.True(t, found, "Upstream should be assigned to pool")
}
