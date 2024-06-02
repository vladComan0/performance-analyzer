package tokens

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog"
	"io"
	"net/http"
	"sync"
	"time"
)

type Credentials struct {
	Username       *string `json:"username"`
	Password       *string `json:"password"`
	BasicAuthToken *string `json:"basic_auth_token"`
}

type Token struct {
	Value     string
	ExpiresAt time.Time
}

type TokenManager struct {
	Token       Token
	Credentials Credentials
	BaseURL     string
	Log         zerolog.Logger
	mu          sync.Mutex
}

func NewTokenManager(credentials Credentials, baseURL string) *TokenManager {
	return &TokenManager{
		Credentials: credentials,
		BaseURL:     baseURL,
	}
}

func (tm *TokenManager) GetToken() (string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if time.Now().After(tm.Token.ExpiresAt) {
		newToken, err := tm.requestNewToken()
		if err != nil {
			return "", err
		}
		tm.Token = newToken
	}
	return tm.Token.Value, nil
}

func (tm *TokenManager) requestNewToken() (Token, error) {
	url := tm.BaseURL + "/v2/oauth/token"

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return Token{}, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if tm.Credentials.Username != nil && tm.Credentials.Password != nil {
		req.SetBasicAuth(*tm.Credentials.Username, *tm.Credentials.Password)
	} else if tm.Credentials.BasicAuthToken != nil {
		req.Header.Set("Authorization", "Basic "+*tm.Credentials.BasicAuthToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Token{}, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			tm.Log.Error().Err(err).Msg("Failed to close response body")
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return Token{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	type response struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}

	var res response
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return Token{}, err
	}

	return Token{
		Value:     res.Token,
		ExpiresAt: res.ExpiresAt,
	}, nil
}
