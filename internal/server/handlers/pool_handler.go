package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/torchlabssoftware/subnetwork_system/internal/db/repository"
	functions "github.com/torchlabssoftware/subnetwork_system/internal/server/functions"
	models "github.com/torchlabssoftware/subnetwork_system/internal/server/models"
)

type PoolHandler struct {
	Queries *repository.Queries
	DB      *sql.DB
}

func NewPoolHandler(queries *repository.Queries, db *sql.DB) *PoolHandler {
	return &PoolHandler{
		Queries: queries,
		DB:      db,
	}
}

func (p *PoolHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/region", p.getRegions)
	r.Post("/region", p.createRegion)
	r.Delete("/region", p.DeleteRegion)
	r.Get("/country", p.getcountries)
	r.Post("/country", p.createCountry)
	r.Delete("/country", p.DeleteCountry)
	r.Get("/upstream", p.getUpstreams)
	r.Post("/upstream", p.createUpstream)
	r.Delete("/upstream", p.DeleteUpstream)
	return r
}

func (p *PoolHandler) getRegions(w http.ResponseWriter, r *http.Request) {

	regions, err := p.Queries.GetRegions(r.Context())
	if err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "server error", err)
		return
	}

	res := []models.GetRegionResponce{}

	for _, region := range regions {
		r := models.GetRegionResponce{
			Id:        region.ID,
			Name:      region.Name,
			CreatedAt: region.CreatedAt,
			UpdatedAt: region.UpdatedAt,
		}

		res = append(res, r)
	}

	functions.RespondwithJSON(w, http.StatusCreated, res)
}

func (p *PoolHandler) createRegion(w http.ResponseWriter, r *http.Request) {
	var req models.CreateRegionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "err in request body", err)
		return
	}

	if req.Name != nil && *req.Name != "" {
		functions.RespondwithError(w, http.StatusBadRequest, "add Request name", fmt.Errorf("no region name"))
		return
	}

	region, err := p.Queries.AddRegion(r.Context(), *req.Name)
	if err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "server error", err)
		return
	}

	res := models.CreateRegionResponce{
		Id:        region.ID,
		Name:      region.Name,
		CreatedAt: region.CreatedAt,
		UpdatedAt: region.UpdatedAt,
	}

	functions.RespondwithJSON(w, http.StatusCreated, res)
}

func (p *PoolHandler) DeleteRegion(w http.ResponseWriter, r *http.Request) {
	var req models.DeleteRegionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "err in request body", err)
		return
	}

	err := p.Queries.DeleteRegion(r.Context(), req.Name)
	if err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "server error", err)
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

	countries, err := p.Queries.GetCountries(r.Context())
	if err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "server error", err)
		return
	}

	res := []models.GetCountryResponce{}

	for _, country := range countries {
		r := models.GetCountryResponce{
			Id:        country.ID,
			Name:      country.Name,
			Code:      country.Code,
			RegionId:  country.RegionID,
			CreatedAt: country.CreatedAt,
			UpdatedAt: country.UpdatedAt,
		}

		res = append(res, r)
	}

	functions.RespondwithJSON(w, http.StatusCreated, res)
}

func (p *PoolHandler) createCountry(w http.ResponseWriter, r *http.Request) {
	var req models.CreateCountryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "err in request body", err)
		return
	}

	if (req.Name == nil && *req.Name == "") || (req.Code == nil && *req.Code == "") || req.RegionId == nil {
		functions.RespondwithError(w, http.StatusBadRequest, "err in request body", fmt.Errorf("err in request body"))
		return
	}

	args := repository.AddCountryParams{
		Name:     *req.Name,
		Code:     *req.Code,
		RegionID: *req.RegionId,
	}

	country, err := p.Queries.AddCountry(r.Context(), args)
	if err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "server error", err)
		return
	}

	res := models.CreateCountryResponce{
		Id:        country.ID,
		Name:      country.Name,
		Code:      country.Code,
		RegionId:  country.RegionID,
		CreatedAt: country.CreatedAt,
		UpdatedAt: country.UpdatedAt,
	}

	functions.RespondwithJSON(w, http.StatusCreated, res)
}

func (p *PoolHandler) DeleteCountry(w http.ResponseWriter, r *http.Request) {
	var req models.DeleteCountryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "err in request body", err)
		return
	}

	err := p.Queries.DeleteCountry(r.Context(), req.Name)
	if err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "server error", err)
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

	upstreams, err := p.Queries.GetUpstreams(r.Context())
	if err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "server error", err)
		return
	}

	res := []models.GetUpstreamResponce{}

	for _, upstream := range upstreams {
		r := models.GetUpstreamResponce{
			Id:               upstream.ID,
			UpstreamProvider: upstream.UpstreamProvider,
			Format:           upstream.Format,
			Domain:           upstream.Domain,
			Port:             int(upstream.Port),
			PoolId:           upstream.PoolID,
			CreatedAt:        upstream.CreatedAt.Time,
			UpdatedAt:        upstream.UpdatedAt.Time,
		}

		res = append(res, r)
	}

	functions.RespondwithJSON(w, http.StatusCreated, res)
}

func (p *PoolHandler) createUpstream(w http.ResponseWriter, r *http.Request) {
	var req models.CreateUpstreamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "err in request body", err)
		return
	}

	if (req.UpstreamProvider == nil && *req.UpstreamProvider == "") ||
		(req.Format == nil && *req.Format == "") ||
		req.Port == nil ||
		(req.Domain == nil && *req.Domain == "") ||
		(req.PoolId == nil) {
		functions.RespondwithError(w, http.StatusBadRequest, "err in request body", fmt.Errorf("err in request body"))
		return
	}

	args := repository.AddUpstreamParams{
		UpstreamProvider: *req.UpstreamProvider,
		Format:           *req.Format,
		Port:             int32(*req.Port),
		Domain:           *req.Domain,
		PoolID:           *req.PoolId,
	}

	upstream, err := p.Queries.AddUpstream(r.Context(), args)
	if err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "server error", err)
		return
	}

	res := models.CreateUpstreamResponce{
		Id:               upstream.ID,
		UpstreamProvider: upstream.UpstreamProvider,
		Format:           upstream.Format,
		Port:             int(upstream.Port),
		Domain:           upstream.Domain,
		PoolId:           upstream.PoolID,
		CreatedAt:        upstream.CreatedAt.Time,
		UpdatedAt:        upstream.UpdatedAt.Time,
	}

	functions.RespondwithJSON(w, http.StatusCreated, res)
}

func (p *PoolHandler) DeleteUpstream(w http.ResponseWriter, r *http.Request) {
	var req models.DeleteUpstreamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "err in request body", err)
		return
	}

	err := p.Queries.DeleteUpstream(r.Context(), req.Id)
	if err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "server error", err)
		return
	}

	res := struct {
		Message string `json:"message"`
	}{
		Message: "deleted",
	}

	functions.RespondwithJSON(w, http.StatusOK, res)
}
