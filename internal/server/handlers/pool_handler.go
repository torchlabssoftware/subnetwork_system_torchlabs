package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/torchlabssoftware/subnetwork_system/internal/db/repository"
	functions "github.com/torchlabssoftware/subnetwork_system/internal/server/functions"
	middleware "github.com/torchlabssoftware/subnetwork_system/internal/server/middleware"
	models "github.com/torchlabssoftware/subnetwork_system/internal/server/models"
	"github.com/torchlabssoftware/subnetwork_system/internal/server/service"
)

type PoolHandler struct {
	Queries *repository.Queries
	DB      *sql.DB
	Service service.PoolService
}

func NewPoolHandler(queries *repository.Queries, db *sql.DB, service service.PoolService) *PoolHandler {
	return &PoolHandler{
		Queries: queries,
		DB:      db,
		Service: service,
	}
}

func (p *PoolHandler) AdminRoutes() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.AdminAuthentication)

	r.Get("/region", p.getRegions)
	r.Post("/region", p.createRegion)
	r.Delete("/region", p.DeleteRegion)
	r.Get("/country", p.getcountries)
	r.Post("/country", p.createCountry)
	r.Delete("/country", p.DeleteCountry)
	r.Get("/upstream", p.getUpstreams)
	r.Post("/upstream", p.createUpstream)
	r.Delete("/upstream", p.deleteUpstream)

	r.Post("/", p.createPool)
	r.Get("/", p.getPools)
	r.Get("/{tag}", p.getPoolByTag)
	r.Put("/{tag}", p.updatePool)
	r.Delete("/{tag}", p.deletePool)
	r.Post("/weight", p.addPoolUpstreamWeight)
	r.Delete("/weight", p.deletePoolUpstreamWeight)
	return r
}

func (p *PoolHandler) getRegions(w http.ResponseWriter, r *http.Request) {

	regions, status, message, err := p.Service.GetRegions(r.Context())
	if err != nil {
		functions.RespondwithError(w, status, message, err)
		return
	}

	functions.RespondwithJSON(w, http.StatusOK, regions)
}

func (p *PoolHandler) createRegion(w http.ResponseWriter, r *http.Request) {
	var req models.CreateRegionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "err in request body", err)
		return
	}

	if req.Name == nil || *req.Name == "" {
		functions.RespondwithError(w, http.StatusBadRequest, "add Request name", fmt.Errorf("no region name"))
		return
	}

	res, status, message, err := p.Service.CreateRegion(r.Context(), req)
	if err != nil {
		functions.RespondwithError(w, status, message, err)
		return
	}

	functions.RespondwithJSON(w, http.StatusCreated, res)
}

func (p *PoolHandler) DeleteRegion(w http.ResponseWriter, r *http.Request) {
	var req models.DeleteRegionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "err in request body", err)
		return
	}

	if req.Name == nil || *req.Name == "" {
		functions.RespondwithError(w, http.StatusBadRequest, "add region name", fmt.Errorf("no region name"))
		return
	}

	code, message, err := p.Service.DeleteRegion(r.Context(), *req.Name)
	if err != nil {
		functions.RespondwithError(w, code, message, err)
		return
	}

	res := struct {
		Message string `json:"message"`
	}{
		Message: "deleted",
	}

	functions.RespondwithJSON(w, http.StatusOK, res)
}

func (p *PoolHandler) getcountries(w http.ResponseWriter, r *http.Request) {
	countries, status, message, err := p.Service.GetCountries(r.Context())
	if err != nil {
		functions.RespondwithError(w, status, message, err)
		return
	}

	functions.RespondwithJSON(w, http.StatusOK, countries)
}

func (p *PoolHandler) createCountry(w http.ResponseWriter, r *http.Request) {
	var req models.CreateCountryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "err in request body", err)
		return
	}

	if (req.Name == nil || *req.Name == "") || (req.Code == nil || *req.Code == "") || req.RegionId == nil {
		functions.RespondwithError(w, http.StatusBadRequest, "err in request body", fmt.Errorf("err in request body"))
		return
	}

	res, status, message, err := p.Service.CreateCountry(r.Context(), req)
	if err != nil {
		functions.RespondwithError(w, status, message, err)
		return
	}

	functions.RespondwithJSON(w, http.StatusCreated, res)
}

func (p *PoolHandler) DeleteCountry(w http.ResponseWriter, r *http.Request) {
	var req models.DeleteCountryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "err in request body", err)
		return
	}

	if req.Name == nil || *req.Name == "" {
		functions.RespondwithError(w, http.StatusBadRequest, "add country name", fmt.Errorf("no region name"))
		return
	}

	code, message, err := p.Service.DeleteCountry(r.Context(), *req.Name)
	if err != nil {
		functions.RespondwithError(w, code, message, err)
		return
	}

	res := struct {
		Message string `json:"message"`
	}{
		Message: "deleted",
	}

	functions.RespondwithJSON(w, http.StatusOK, res)
}

func (p *PoolHandler) getUpstreams(w http.ResponseWriter, r *http.Request) {
	upstreams, status, message, err := p.Service.GetUpstreams(r.Context())
	if err != nil {
		functions.RespondwithError(w, status, message, err)
		return
	}

	functions.RespondwithJSON(w, http.StatusOK, upstreams)
}

func (p *PoolHandler) createUpstream(w http.ResponseWriter, r *http.Request) {
	var req models.CreateUpstreamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "err in request body", err)
		return
	}

	if (req.UpstreamProvider == nil || *req.UpstreamProvider == "") ||
		(req.Format == nil || *req.Format == "") ||
		(req.Tag == nil || *req.Tag == "") ||
		req.Port == nil ||
		(req.Domain == nil || *req.Domain == "") {
		functions.RespondwithError(w, http.StatusBadRequest, "err in request body", fmt.Errorf("err in request body"))
		return
	}

	res, status, message, err := p.Service.CreateUpstream(r.Context(), req)
	if err != nil {
		functions.RespondwithError(w, status, message, err)
		return
	}

	functions.RespondwithJSON(w, http.StatusCreated, res)
}

func (p *PoolHandler) deleteUpstream(w http.ResponseWriter, r *http.Request) {
	var req models.DeleteUpstreamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "err in request body", err)
		return
	}

	if req.Tag == nil || *req.Tag == "" {
		functions.RespondwithError(w, http.StatusBadRequest, "tag is required", fmt.Errorf("tag is required"))
		return
	}

	code, message, err := p.Service.DeleteUpstream(r.Context(), *req.Tag)
	if err != nil {
		functions.RespondwithError(w, code, message, err)
		return
	}

	res := struct {
		Message string `json:"message"`
	}{
		Message: "deleted",
	}

	functions.RespondwithJSON(w, http.StatusOK, res)
}

func (p *PoolHandler) createPool(w http.ResponseWriter, r *http.Request) {

	var req models.CreatePoolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "err in request body", err)
		return
	}

	if req.Tag == nil || *req.Tag == "" {
		functions.RespondwithError(w, http.StatusBadRequest, "tag is required", fmt.Errorf("tag is required"))
		return
	}
	if req.RegionId == nil {
		functions.RespondwithError(w, http.StatusBadRequest, "region id is required", fmt.Errorf("region id is required"))
		return
	}

	if req.Subdomain == nil || *req.Subdomain == "" {
		functions.RespondwithError(w, http.StatusBadRequest, "subdomain is required", fmt.Errorf("subdomain is required"))
		return
	}
	if req.Port == nil {
		functions.RespondwithError(w, http.StatusBadRequest, "port is required", fmt.Errorf("port is required"))
		return
	}
	if req.UpStreams == nil {
		req.UpStreams = &[]models.CreateUpstreamWeightRequest{}
	}

	res, status, message, err := p.Service.CreatePool(r.Context(), req)
	if err != nil {
		functions.RespondwithError(w, status, message, err)
		return
	}

	functions.RespondwithJSON(w, http.StatusCreated, res)

}

func (p *PoolHandler) getPools(w http.ResponseWriter, r *http.Request) {
	response, status, message, err := p.Service.GetPools(r.Context())
	if err != nil {
		functions.RespondwithError(w, status, message, err)
		return
	}

	functions.RespondwithJSON(w, http.StatusOK, response)
}

func (p *PoolHandler) getPoolByTag(w http.ResponseWriter, r *http.Request) {
	tag := chi.URLParam(r, "tag")
	if tag == "" {
		functions.RespondwithError(w, http.StatusBadRequest, "Tag is required", fmt.Errorf("missing tag param"))
		return
	}

	poolResponse, status, message, err := p.Service.GetPoolByTag(r.Context(), tag)
	if err != nil {
		functions.RespondwithError(w, status, message, err)
		return
	}

	functions.RespondwithJSON(w, http.StatusOK, poolResponse)
}

func (p *PoolHandler) updatePool(w http.ResponseWriter, r *http.Request) {
	tagStr := chi.URLParam(r, "tag")
	if tagStr == "" {
		functions.RespondwithError(w, http.StatusBadRequest, "Pool Tag is required", fmt.Errorf("missing tag param"))
		return
	}

	var req models.UpdatePoolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	res, status, message, err := p.Service.UpdatePool(r.Context(), tagStr, req)
	if err != nil {
		functions.RespondwithError(w, status, message, err)
		return
	}

	functions.RespondwithJSON(w, http.StatusOK, res)
}

func (p *PoolHandler) deletePool(w http.ResponseWriter, r *http.Request) {
	tag := chi.URLParam(r, "tag")
	if tag == "" {
		functions.RespondwithError(w, http.StatusBadRequest, "Tag is required", fmt.Errorf("missing tag param"))
		return
	}

	code, message, err := p.Service.DeletePool(r.Context(), tag)
	if err != nil {
		functions.RespondwithError(w, code, message, err)
		return
	}

	res := struct {
		Message string `json:"message"`
	}{
		Message: message,
	}

	functions.RespondwithJSON(w, http.StatusOK, res)
}

func (p *PoolHandler) addPoolUpstreamWeight(w http.ResponseWriter, r *http.Request) {
	var req models.AddPoolUpstreamWeightRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if (req.PoolTag == nil || *req.PoolTag == "") || (req.UpstreamTag == nil || *req.UpstreamTag == "") || (req.Weight == nil || *req.Weight == 0) {
		functions.RespondwithError(w, http.StatusBadRequest, "Pool Tag, Upstream Tag and Weight are required", fmt.Errorf("missing fields"))
		return
	}

	code, message, err := p.Service.AddPoolUpstreamWeight(r.Context(), req)
	if err != nil {
		functions.RespondwithError(w, code, message, err)
		return
	}

	res := struct {
		Message string `json:"message"`
	}{
		Message: message,
	}

	functions.RespondwithJSON(w, http.StatusCreated, res)
}

func (p *PoolHandler) deletePoolUpstreamWeight(w http.ResponseWriter, r *http.Request) {
	var req models.DeletePoolUpstreamWeightRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if (req.PoolTag == nil || *req.PoolTag == "") || (req.UpstreamTag == nil || *req.UpstreamTag == "") {
		functions.RespondwithError(w, http.StatusBadRequest, "Pool Tag and Upstream Tag are required", fmt.Errorf("missing fields"))
		return
	}

	code, message, err := p.Service.DeletePoolUpstreamWeight(r.Context(), req)
	if err != nil {
		functions.RespondwithError(w, code, message, err)
		return
	}

	res := struct {
		Message string `json:"message"`
	}{
		Message: message,
	}

	functions.RespondwithJSON(w, http.StatusOK, res)
}
