package server

import (
	"time"

	"github.com/google/uuid"
)

type GetRegionResponce struct {
	Id        uuid.UUID `json:"id"`
	Name      string    `json:"name,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateRegionRequest struct {
	Name *string `json:"name"`
}

type CreateRegionResponce struct {
	Id        uuid.UUID `json:"id"`
	Name      string    `json:"name,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type DeleteRegionRequest struct {
	Name *string `json:"name"`
}

type GetCountryResponce struct {
	Id        uuid.UUID `json:"id"`
	Name      string    `json:"name,omitempty"`
	Code      string    `json:"code,omitempty"`
	RegionId  uuid.UUID `json:"region_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateCountryRequest struct {
	Name     *string    `json:"name"`
	Code     *string    `json:"code"`
	RegionId *uuid.UUID `json:"region_id"`
}

type CreateCountryResponce struct {
	Id        uuid.UUID `json:"id"`
	Name      string    `json:"name,omitempty"`
	Code      string    `json:"code,omitempty"`
	RegionId  uuid.UUID `json:"region_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type DeleteCountryRequest struct {
	Name *string `json:"name"`
}

type GetUpstreamResponce struct {
	Id               uuid.UUID `json:"id"`
	Tag              string    `json:"tag"`
	UpstreamProvider string    `json:"upstream_provider"`
	Format           string    `json:"format"`
	Port             int       `json:"port"`
	Domain           string    `json:"domain"`
	CreatedAt        time.Time `json:"created_at"`
}

type CreateUpstreamRequest struct {
	Tag              *string `json:"tag"`
	UpstreamProvider *string `json:"upstream_provider"`
	Format           *string `json:"format"`
	Port             *int    `json:"port"`
	Domain           *string `json:"domain"`
}

type CreateUpstreamResponce struct {
	Id               uuid.UUID `json:"id"`
	Tag              string    `json:"tag"`
	UpstreamProvider string    `json:"upstream_provider"`
	Format           string    `json:"format"`
	Port             int       `json:"port"`
	Domain           string    `json:"domain"`
	CreatedAt        time.Time `json:"created_at"`
}

type DeleteUpstreamRequest struct {
	Tag *string `json:"tag"`
}

type CreatePoolRequest struct {
	Tag       *string                        `json:"tag"`
	RegionId  *uuid.UUID                     `json:"region_id"`
	Subdomain *string                        `json:"subdomain"`
	Port      *int32                         `json:"port"`
	UpStreams *[]CreateUpstreamWeightRequest `json:"upstreams"`
}

type CreateUpstreamWeightRequest struct {
	UpstreamTag *string `json:"upstream_tag"`
	Weight      *int32  `json:"weight"`
}

type CreateUpstreamWeightResponce struct {
	UpstreamTag string `json:"upstream_tag,omitempty"`
	Weight      int32  `json:"weight,omitempty"`
}

type CreatePoolResponce struct {
	Id        uuid.UUID                      `json:"id,omitempty"`
	Tag       string                         `json:"tag,omitempty"`
	RegionId  uuid.UUID                      `json:"region_id,omitempty"`
	Subdomain string                         `json:"subdomain,omitempty"`
	Port      int32                          `json:"port,omitempty"`
	UpStreams []CreateUpstreamWeightResponce `json:"upstreams,omitempty"`
	CreatedAt time.Time                      `json:"created_at,omitempty"`
	UpdatedAt time.Time                      `json:"updated_at,omitempty"`
}

type PoolUpstream struct {
	Tag    string `json:"tag"`
	Format string `json:"format"`
	Port   int32  `json:"port"`
	Domain string `json:"domain"`
}

type GetPoolsResponse struct {
	Id        uuid.UUID      `json:"id,omitempty"`
	Tag       string         `json:"tag,omitempty"`
	Subdomain string         `json:"subdomain,omitempty"`
	Port      int32          `json:"port,omitempty"`
	Upstreams []PoolUpstream `json:"upstreams,omitempty"`
}

type UpdatePoolRequest struct {
	RegionId  *uuid.UUID `json:"region_id"`
	Subdomain *string    `json:"subdomain"`
	Port      *int32     `json:"port"`
}

type AddPoolUpstreamWeightRequest struct {
	PoolTag     *string `json:"pool_tag"`
	UpstreamTag *string `json:"upstream_tag"`
	Weight      *int32  `json:"weight"`
}

type DeletePoolUpstreamWeightRequest struct {
	PoolTag     *string `json:"pool_tag"`
	UpstreamTag *string `json:"upstream_tag"`
}
