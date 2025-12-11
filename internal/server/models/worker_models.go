package server

type AddWorkerRequest struct {
	Name       *string `json:"name"`
	RegionName *string `json:"region_name"`
	IPAddress  *string `json:"ip_address"`
}

type AddWorkerResponse struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	RegionName string   `json:"region_name"`
	IpAddress  string   `json:"ip_address"`
	Status     string   `json:"status"`
	LastSeen   string   `json:"last_seen"`
	CreatedAt  string   `json:"created_at"`
	UpdatedAt  string   `json:"updated_at"`
	Domains    []string `json:"domains,omitempty"`
}
