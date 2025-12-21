package service

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/torchlabssoftware/subnetwork_system/internal/db/repository"
	models "github.com/torchlabssoftware/subnetwork_system/internal/server/models"
)

type PoolService interface {
	GetRegions(ctx context.Context) ([]models.GetRegionResponce, int, string, error)
	CreateRegion(ctx context.Context, req models.CreateRegionRequest) (models.CreateRegionResponce, int, string, error)
	DeleteRegion(ctx context.Context, name string) (int, string, error)
	GetCountries(ctx context.Context) ([]models.GetCountryResponce, int, string, error)
	CreateCountry(ctx context.Context, req models.CreateCountryRequest) (models.CreateCountryResponce, int, string, error)
	DeleteCountry(ctx context.Context, name string) (int, string, error)
	GetUpstreams(ctx context.Context) ([]models.GetUpstreamResponce, int, string, error)
	CreateUpstream(ctx context.Context, req models.CreateUpstreamRequest) (models.CreateUpstreamResponce, int, string, error)
	DeleteUpstream(ctx context.Context, tag string) (int, string, error)
	GetPools(ctx context.Context) ([]models.GetPoolsResponse, int, string, error)
	GetPoolByTag(ctx context.Context, tag string) (*models.GetPoolsResponse, int, string, error)
	CreatePool(ctx context.Context, req models.CreatePoolRequest) (models.CreatePoolResponce, int, string, error)
	UpdatePool(ctx context.Context, tag string, req models.UpdatePoolRequest) (models.CreatePoolResponce, int, string, error)
	DeletePool(ctx context.Context, tag string) (int, string, error)
	AddPoolUpstreamWeight(ctx context.Context, req models.AddPoolUpstreamWeightRequest) (int, string, error)
	DeletePoolUpstreamWeight(ctx context.Context, req models.DeletePoolUpstreamWeightRequest) (int, string, error)
}

type PoolServiceImpl struct {
	Queries *repository.Queries
	DB      *sql.DB
}

func NewPoolService(queries *repository.Queries, db *sql.DB) PoolService {
	return &PoolServiceImpl{
		Queries: queries,
		DB:      db,
	}
}

func (s *PoolServiceImpl) GetRegions(ctx context.Context) ([]models.GetRegionResponce, int, string, error) {
	regions, err := s.Queries.GetRegions(ctx)
	if err != nil {
		return nil, http.StatusInternalServerError, "failed to get regions", err
	}

	res := []models.GetRegionResponce{}

	for _, region := range regions {
		r := models.GetRegionResponce{
			Id:        region.ID,
			Name:      region.Name,
			CreatedAt: region.CreatedAt,
		}

		res = append(res, r)
	}

	return res, http.StatusOK, "", nil
}

func (s *PoolServiceImpl) CreateRegion(ctx context.Context, req models.CreateRegionRequest) (models.CreateRegionResponce, int, string, error) {
	region, err := s.Queries.AddRegion(ctx, *req.Name)
	if err != nil {
		return models.CreateRegionResponce{}, http.StatusInternalServerError, "failed to create region", err
	}

	res := models.CreateRegionResponce{
		Id:        region.ID,
		Name:      region.Name,
		CreatedAt: region.CreatedAt,
	}

	return res, http.StatusCreated, "region created", nil
}

func (s *PoolServiceImpl) DeleteRegion(ctx context.Context, name string) (int, string, error) {
	if err := s.Queries.DeleteRegion(ctx, name); err != nil {
		return http.StatusInternalServerError, "failed to delete region", err
	}
	return http.StatusOK, "region deleted", nil
}

func (s *PoolServiceImpl) GetCountries(ctx context.Context) ([]models.GetCountryResponce, int, string, error) {
	countries, err := s.Queries.GetCountries(ctx)
	if err != nil {
		return nil, http.StatusInternalServerError, "failed to get countries", err
	}

	res := []models.GetCountryResponce{}

	for _, country := range countries {
		r := models.GetCountryResponce{
			Id:        country.ID,
			Name:      country.Name,
			Code:      country.Code,
			RegionId:  country.RegionID,
			CreatedAt: country.CreatedAt,
		}

		res = append(res, r)
	}

	return res, http.StatusOK, "", nil
}

func (s *PoolServiceImpl) CreateCountry(ctx context.Context, req models.CreateCountryRequest) (models.CreateCountryResponce, int, string, error) {
	args := repository.AddCountryParams{
		Name:     *req.Name,
		Code:     *req.Code,
		RegionID: *req.RegionId,
	}

	country, err := s.Queries.AddCountry(ctx, args)
	if err != nil {
		return models.CreateCountryResponce{}, http.StatusInternalServerError, "failed to create country", err
	}

	res := models.CreateCountryResponce{
		Id:        country.ID,
		Name:      country.Name,
		Code:      country.Code,
		RegionId:  country.RegionID,
		CreatedAt: country.CreatedAt,
	}

	return res, http.StatusCreated, "country created", nil
}

func (s *PoolServiceImpl) DeleteCountry(ctx context.Context, name string) (int, string, error) {
	if err := s.Queries.DeleteCountry(ctx, name); err != nil {
		return http.StatusInternalServerError, "failed to delete country", err
	}
	return http.StatusOK, "country deleted", nil
}

func (s *PoolServiceImpl) GetUpstreams(ctx context.Context) ([]models.GetUpstreamResponce, int, string, error) {
	upstreams, err := s.Queries.GetUpstreams(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, http.StatusNotFound, "upstreams not found", nil
		}
		return nil, http.StatusInternalServerError, "failed to get upstreams", err
	}

	res := []models.GetUpstreamResponce{}

	for _, upstream := range upstreams {
		r := models.GetUpstreamResponce{
			Id:               upstream.ID,
			Tag:              upstream.Tag,
			UpstreamProvider: upstream.UpstreamProvider,
			Format:           upstream.Format,
			Domain:           upstream.Domain,
			Port:             int(upstream.Port),
			CreatedAt:        upstream.CreatedAt,
		}

		res = append(res, r)
	}

	return res, http.StatusOK, "", nil
}

func (s *PoolServiceImpl) CreateUpstream(ctx context.Context, req models.CreateUpstreamRequest) (models.CreateUpstreamResponce, int, string, error) {

	args := repository.AddUpstreamParams{
		Tag:              *req.Tag,
		UpstreamProvider: *req.UpstreamProvider,
		Format:           *req.Format,
		Port:             int32(*req.Port),
		Domain:           *req.Domain,
	}

	upstream, err := s.Queries.AddUpstream(ctx, args)
	if err != nil {
		return models.CreateUpstreamResponce{}, http.StatusInternalServerError, "failed to create upstream", err
	}

	res := models.CreateUpstreamResponce{
		Id:               upstream.ID,
		Tag:              upstream.Tag,
		UpstreamProvider: upstream.UpstreamProvider,
		Format:           upstream.Format,
		Port:             int(upstream.Port),
		Domain:           upstream.Domain,
		CreatedAt:        upstream.CreatedAt,
	}

	return res, http.StatusCreated, "upstream created", nil
}

func (s *PoolServiceImpl) DeleteUpstream(ctx context.Context, tag string) (int, string, error) {
	if err := s.Queries.DeleteUpstreamByTag(ctx, tag); err != nil {
		return http.StatusInternalServerError, "failed to delete upstream", err
	}
	return http.StatusOK, "upstream deleted", nil
}

func (s *PoolServiceImpl) GetPools(ctx context.Context) ([]models.GetPoolsResponse, int, string, error) {
	rows, err := s.Queries.ListPoolsWithUpstreams(ctx)
	if err != nil {
		return nil, http.StatusInternalServerError, "Failed to fetch pools", err
	}

	if len(rows) <= 0 {
		return nil, http.StatusNotFound, "pools not found", nil
	}

	poolMap := make(map[uuid.UUID]*models.GetPoolsResponse)

	var orderedPools []*models.GetPoolsResponse

	for _, row := range rows {
		pool, exists := poolMap[row.PoolID]
		if !exists {
			pool = &models.GetPoolsResponse{
				Id:        row.PoolID,
				Tag:       row.PoolTag,
				Subdomain: row.PoolSubdomain,
				Port:      row.PoolPort,
				Upstreams: []models.PoolUpstream{},
			}
			poolMap[row.PoolID] = pool
			orderedPools = append(orderedPools, pool)
		}

		if row.UpstreamTag.Valid {
			pool.Upstreams = append(pool.Upstreams, models.PoolUpstream{
				Tag:    row.UpstreamTag.String,
				Format: row.UpstreamFormat.String,
				Port:   row.UpstreamPort.Int32,
				Domain: row.UpstreamDomain.String,
			})
		}
	}

	response := make([]models.GetPoolsResponse, 0, len(orderedPools))
	for _, pool := range orderedPools {
		response = append(response, *pool)
	}

	return response, http.StatusOK, "", nil
}

func (s *PoolServiceImpl) GetPoolByTag(ctx context.Context, tag string) (*models.GetPoolsResponse, int, string, error) {
	rows, err := s.Queries.GetPoolByTagWithUpstreams(ctx, tag)
	if err != nil {
		return nil, http.StatusInternalServerError, "Failed to fetch pool", err
	}

	if len(rows) == 0 {
		return nil, http.StatusNotFound, "Pool not found", fmt.Errorf("pool not found")
	}

	var poolResponse *models.GetPoolsResponse

	for _, row := range rows {
		if poolResponse == nil {
			poolResponse = &models.GetPoolsResponse{
				Id:        row.PoolID,
				Tag:       row.PoolTag,
				Subdomain: row.PoolSubdomain,
				Port:      row.PoolPort,
				Upstreams: []models.PoolUpstream{},
			}
		}

		if row.UpstreamTag.Valid {
			poolResponse.Upstreams = append(poolResponse.Upstreams, models.PoolUpstream{
				Tag:    row.UpstreamTag.String,
				Format: row.UpstreamFormat.String,
				Port:   row.UpstreamPort.Int32,
				Domain: row.UpstreamDomain.String,
			})
		}
	}

	return poolResponse, http.StatusOK, "", nil
}

func (s *PoolServiceImpl) CreatePool(ctx context.Context, req models.CreatePoolRequest) (models.CreatePoolResponce, int, string, error) {
	//begin transaction
	tx, err := s.DB.Begin()
	if err != nil {
		return models.CreatePoolResponce{}, http.StatusInternalServerError, "failed to create user", err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	qtx := s.Queries.WithTx(tx)

	args := repository.InsetPoolParams{
		Tag:       *req.Tag,
		RegionID:  *req.RegionId,
		Subdomain: *req.Subdomain,
		Port:      *req.Port,
	}

	pool, err := qtx.InsetPool(ctx, args)
	if err != nil {
		return models.CreatePoolResponce{}, http.StatusBadRequest, "server error", err
	}

	weights := []int32{}
	upstreamTags := []string{}

	for _, upstreams := range *req.UpStreams {
		weights = append(weights, *upstreams.Weight)
		upstreamTags = append(upstreamTags, *upstreams.UpstreamTag)
	}

	weightArgs := repository.InsertPoolUpstreamWeightParams{
		PoolID:  pool.ID,
		Column2: weights,
		Column3: upstreamTags,
	}

	poolUpstreamWeights, err := qtx.InsertPoolUpstreamWeight(ctx, weightArgs)
	if err != nil {
		return models.CreatePoolResponce{}, http.StatusBadRequest, "server error", err
	}

	if err := tx.Commit(); err != nil {
		return models.CreatePoolResponce{}, http.StatusInternalServerError, "failed to create pool", err
	}

	upstreamsRes := []models.CreateUpstreamWeightResponce{}

	for i, puw := range poolUpstreamWeights {
		upstreamRes := models.CreateUpstreamWeightResponce{
			UpstreamTag: upstreamTags[i],
			Weight:      puw.Weight,
		}
		upstreamsRes = append(upstreamsRes, upstreamRes)
	}
	res := models.CreatePoolResponce{
		Id:        pool.ID,
		Tag:       pool.Tag,
		RegionId:  pool.RegionID,
		Subdomain: pool.Subdomain,
		Port:      pool.Port,
		UpStreams: upstreamsRes,
		CreatedAt: pool.CreatedAt,
		UpdatedAt: pool.UpdatedAt,
	}

	return res, http.StatusCreated, "pool created", nil
}

func (s *PoolServiceImpl) UpdatePool(ctx context.Context, tag string, req models.UpdatePoolRequest) (models.CreatePoolResponce, int, string, error) {
	regionId := uuid.NullUUID{Valid: false}
	if req.RegionId != nil {
		regionId = uuid.NullUUID{UUID: *req.RegionId, Valid: true}
	}

	subdomain := sql.NullString{Valid: false}
	if req.Subdomain != nil {
		subdomain = sql.NullString{String: *req.Subdomain, Valid: true}
	}

	port := sql.NullInt32{Valid: false}
	if req.Port != nil {
		port = sql.NullInt32{Int32: *req.Port, Valid: true}
	}

	args := repository.UpdatePoolParams{
		Tag:       tag,
		RegionID:  regionId,
		Subdomain: subdomain,
		Port:      port,
	}

	updatedPool, err := s.Queries.UpdatePool(ctx, args)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.CreatePoolResponce{}, http.StatusNotFound, "Pool not found", err
		}
		return models.CreatePoolResponce{}, http.StatusInternalServerError, "Failed to update pool", err
	}

	res := models.CreatePoolResponce{
		Id:        updatedPool.ID,
		Tag:       updatedPool.Tag,
		RegionId:  updatedPool.RegionID,
		Subdomain: updatedPool.Subdomain,
		Port:      updatedPool.Port,
		CreatedAt: updatedPool.CreatedAt,
		UpdatedAt: updatedPool.UpdatedAt,
	}

	return res, http.StatusOK, "pool updated", nil
}

func (s *PoolServiceImpl) DeletePool(ctx context.Context, tag string) (int, string, error) {
	result, err := s.Queries.DeletePool(ctx, tag)
	if err != nil {
		return http.StatusInternalServerError, "Failed to delete pool", err
	}
	rowsAffected, _ := result.RowsAffected()
	log.Println(rowsAffected)
	if rowsAffected == 0 {
		return http.StatusNotFound, "Nothing deleted", nil
	}
	return http.StatusOK, "deleted", nil
}

func (s *PoolServiceImpl) AddPoolUpstreamWeight(ctx context.Context, req models.AddPoolUpstreamWeightRequest) (int, string, error) {
	args := repository.AddPoolUpstreamWeightParams{
		Tag:    *req.PoolTag,
		Tag_2:  *req.UpstreamTag,
		Weight: *req.Weight,
	}

	_, err := s.Queries.AddPoolUpstreamWeight(ctx, args)
	if err != nil {
		return http.StatusInternalServerError, "Failed to add upstream weight", err
	}
	return http.StatusCreated, "added", nil
}

func (s *PoolServiceImpl) DeletePoolUpstreamWeight(ctx context.Context, req models.DeletePoolUpstreamWeightRequest) (int, string, error) {
	args := repository.DeletePoolUpstreamWeightParams{
		Tag:   *req.PoolTag,
		Tag_2: *req.UpstreamTag,
	}

	result, err := s.Queries.DeletePoolUpstreamWeight(ctx, args)
	if err != nil {
		return http.StatusInternalServerError, "Failed to delete upstream weight", err
	}
	rowsAffected, _ := result.RowsAffected()
	log.Println(rowsAffected)
	if rowsAffected == 0 {
		return http.StatusNotFound, "Nothing deleted", nil
	}
	return http.StatusOK, "deleted", nil
}
