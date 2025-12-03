package server

type CreateUserRequest struct {
	Email       *string   `json:"email"`
	DataLimit   *int64    `json:"data_limit"`
	AllowPools  *[]string `json:"allow_pools"`
	IpWhiteList *[]string `json:"ip_whitelist"`
}

type CreateUserResponce struct {
	Username    *string   `json:"username,omitempty"`
	Password    *string   `json:"password,omitempty"`
	DataLimit   *int64    `json:"data_limit,omitempty"`
	IpWhitelist *[]string `json:"ip_whitelist,omitempty"`
	AllowPools  *[]string `json:"allow_pools,omitempty"`
}
