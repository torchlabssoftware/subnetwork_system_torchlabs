package server

import (
	"time"

	"github.com/google/uuid"
)

type CreateUserRequest struct {
	AllowPools  *[]PoolDataStat `json:"allow_pools"`
	IpWhiteList *[]string       `json:"ip_whitelist"`
}

type CreateUserResponce struct {
	Id          uuid.UUID `json:"id,omitempty"`
	Username    string    `json:"username,omitempty"`
	Password    string    `json:"password,omitempty"`
	IpWhitelist []string  `json:"ip_whitelist,omitempty"`
	AllowPools  []string  `json:"allow_pools,omitempty"`
}

type GetUserByIdResponce struct {
	Id          uuid.UUID `json:"id,omitempty"`
	Username    string    `json:"username,omitempty"`
	Password    string    `json:"password,omitempty"`
	Status      string    `json:"status,omitempty"`
	UserPool    []string  `json:"user_pool,omitempty"`
	IpWhitelist []string  `json:"ip_whitelist,omitempty"`
	Created_at  time.Time `json:"created_at,omitempty"`
	Updated_at  time.Time `json:"updated_at,omitempty"`
}

type UpdateUserRequest struct {
	Status *string `json:"status"`
}

type UpdateUserResponce struct {
	Id        uuid.UUID `json:"id,omitempty"`
	Status    string    `json:"status,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type GetDatausageReponce struct {
	DataLimit int64  `json:"data_limit"`
	DataUsage int64  `json:"data_usage"`
	PoolTag   string `json:"pool_tag"`
}

type PoolDataStat struct {
	Pool      string `json:"pool"`
	DataLimit int64  `json:"data_limit"`
	DataUsage int64  `json:"data_usage"`
}

type GetUserPoolResponce struct {
	UserId uuid.UUID      `json:"user_id"`
	Pools  []PoolDataStat `json:"pools"`
}

type AddUserPoolRequest struct {
	UserPool []PoolDataStat `json:"user_pool"`
}

type AddUserPoolResponce struct {
	UserId   uuid.UUID      `json:"user_id,omitempty"`
	UserPool []PoolDataStat `json:"user_pool,omitempty"`
}

type DeleteUserpoolRequest struct {
	UserPool []string `json:"user_pool"`
}

type GetUserIpwhitelistResponce struct {
	UserId      uuid.UUID `json:"user_id"`
	IpWhitelist []string  `json:"ip_whitelist"`
}

type AddUserIpwhitelistRequest struct {
	IpWhitelist []string `json:"ip_whitelist"`
}

type AddUserIpwhitelistResponce struct {
	UserId      uuid.UUID `json:"user_id,omitempty"`
	IpWhitelist []string  `json:"ip_whitelist,omitempty"`
}

type DeleteUserIpwhitelistRequest struct {
	IpCidr []string `json:"ip_cidr"`
}

type GenerateproxyStringRequest struct {
	UserId      *uuid.UUID `json:"user_id"`
	CountryCode *string    `json:"country_code"`
	PoolGroup   *string    `json:"pool_group"`
	ProxyType   *string    `json:"proxy_type"`
	IsSticky    *bool      `json:"is_sticky"`
	Amount      *int       `json:"amount"`
	Format      *string    `json:"format"`
}
