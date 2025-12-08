package server

import (
	"time"

	"github.com/google/uuid"
)

type CreateUserRequest struct {
	Email       *string   `json:"email"`
	DataLimit   *int64    `json:"data_limit"`
	AllowPools  *[]string `json:"allow_pools"`
	IpWhiteList *[]string `json:"ip_whitelist"`
}

type CreateUserResponce struct {
	Id          uuid.UUID `json:"id,omitempty"`
	Username    string    `json:"username,omitempty"`
	Password    string    `json:"password,omitempty"`
	DataLimit   int64     `json:"data_limit,omitempty"`
	IpWhitelist []string  `json:"ip_whitelist,omitempty"`
	AllowPools  []string  `json:"allow_pools,omitempty"`
}

type GetUserByIdResponce struct {
	Id          uuid.UUID `json:"id,omitempty"`
	Email       string    `json:"email,omitempty"`
	Username    string    `json:"username,omitempty"`
	Password    string    `json:"password,omitempty"`
	Data_limit  int64     `json:"data_limit,omitempty"`
	Data_usage  int64     `json:"data_usage"`
	Status      string    `json:"status,omitempty"`
	UserPool    []string  `json:"user_pool,omitempty"`
	IpWhitelist []string  `json:"ip_whitelist,omitempty"`
	Created_at  time.Time `json:"created_at,omitempty"`
	Updated_at  time.Time `json:"updated_at,omitempty"`
}

type UpdateUserRequest struct {
	Email     *string `json:"email"`
	DataLimit *int64  `json:"data_limit"`
	Status    *string `json:"status"`
}

type UpdateUserResponce struct {
	Id        uuid.UUID `json:"id,omitempty"`
	Email     string    `json:"email,omitempty"`
	DataLimit int64     `json:"data_limit,omitempty"`
	Status    string    `json:"status,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type GetDatausageReponce struct {
	DataLimit int64 `json:"data_limit"`
	DataUsage int64 `json:"data_usage"`
}

type GetUserPoolResponce struct {
	UserId uuid.UUID `json:"user_id"`
	Pools  []string  `json:"pools"`
}

type AddUserPoolRequest struct {
	UserPool []string `json:"user_pool"`
}

type AddUserPoolResponce struct {
	UserId   uuid.UUID `json:"user_id,omitempty"`
	UserPool []string  `json:"user_pool,omitempty"`
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
