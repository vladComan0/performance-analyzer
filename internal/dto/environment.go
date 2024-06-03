package dto

type CreateEnvironmentInput struct {
	Name          string  `json:"name"`
	Endpoint      string  `json:"endpoint"`
	TokenEndpoint *string `json:"token_endpoint"`
	Username      *string `json:"username"`
	Password      *string `json:"password"`
	Disabled      *bool   `json:"disabled"`
}

type UpdateEnvironmentInput struct {
	Name          *string `json:"name"`
	Endpoint      *string `json:"endpoint"`
	TokenEndpoint *string `json:"token"`
	Username      *string `json:"username"`
	Password      *string `json:"password"`
	Disabled      *bool   `json:"disabled"`
}
