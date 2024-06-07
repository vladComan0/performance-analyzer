package tokens

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog"
	"net/http"
	"net/url"
	"strings"
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
	ExpiresIn time.Duration
	FetchedAt time.Time
}

type TokenManager struct {
	Token       Token
	Credentials Credentials
	BaseURL     string
	Log         zerolog.Logger
	mu          sync.Mutex
}

func NewTokenManager(credentials Credentials, baseURL string, log zerolog.Logger) *TokenManager {
	return &TokenManager{
		Credentials: credentials,
		BaseURL:     baseURL,
		Log:         log,
	}
}

func (tm *TokenManager) GetToken() (string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if time.Now().After(tm.Token.FetchedAt.Add(tm.Token.ExpiresIn)) {
		tm.Log.Debug().Msg("Token expired, requesting new one")
		newToken, err := tm.requestNewToken()
		if err != nil {
			return "", err
		}
		tm.Token = newToken
	}
	return tm.Token.Value, nil
}

func (tm *TokenManager) requestNewToken() (Token, error) {
	urlStr := tm.BaseURL + "/v2/oauth/token"
	data := url.Values{}
	data.Set("grant_type", "password")
	data.Set("username", *tm.Credentials.Username)
	data.Set("password", *tm.Credentials.Password)

	req, err := http.NewRequest("POST", urlStr, strings.NewReader(data.Encode()))
	if err != nil {
		return Token{}, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+*tm.Credentials.BasicAuthToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Token{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Token{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	type response struct {
		Token     string        `json:"access_token"`
		ExpiresIn time.Duration `json:"expires_in"`
	}

	var res response
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		tm.Log.Error().Err(err).Msg("Error decoding response")
	}

	expiresIn := res.ExpiresIn * time.Second
	tm.Log.Debug().Msgf("Fetched new token, expires in: %s", expiresIn)

	return Token{
		Value:     res.Token,
		ExpiresIn: expiresIn,
		FetchedAt: time.Now(),
	}, nil
}
