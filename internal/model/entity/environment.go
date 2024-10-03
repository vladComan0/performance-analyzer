package entity

import (
	"time"
)

type Environment struct {
	ID             int       `json:"id"`
	Name           string    `json:"name"`
	Endpoint       string    `json:"endpoint"`
	TokenEndpoint  string    `json:"token_endpoint,omitempty"`
	Username       string    `json:"username,omitempty"`
	Password       string    `json:"password,omitempty"`
	BasicAuthToken string    `json:"basic_auth_token,omitempty"`
	Disabled       bool      `json:"disabled,omitempty"`
	CreatedAt      time.Time `json:"-"`
}

// NewEnvironment creates a new Environment with the given options.
func NewEnvironment(name, endpoint string, options ...EnvironmentOption) *Environment {
	environment := &Environment{
		Name:     name,
		Endpoint: endpoint,
	}

	for _, opt := range options {
		opt(environment)
	}

	return environment
}
