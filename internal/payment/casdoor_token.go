package payment

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

type cachedToken struct {
	accessToken string
	expiresAt   time.Time
}

var (
	tokenMu      sync.Mutex
	currentToken *cachedToken
)

type casdoorTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

// GetPaymentServiceToken returns a cached machine-to-machine JWT from Casdoor,
// fetching and caching a new one when none is cached or the cached one has expired.
func GetPaymentServiceToken() (string, error) {
	tokenMu.Lock()
	defer tokenMu.Unlock()

	if currentToken != nil && time.Now().Before(currentToken.expiresAt) {
		return currentToken.accessToken, nil
	}

	token, err := fetchCasdoorToken()
	if err != nil {
		return "", err
	}

	currentToken = token
	return token.accessToken, nil
}

func fetchCasdoorToken() (*cachedToken, error) {
	endpoint := os.Getenv("CASDOOR_ENDPOINT")
	clientID := os.Getenv("CASDOOR_CLIENT_ID")
	clientSecret := os.Getenv("CASDOOR_CLIENT_SECRET")

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("scope", "openid")

	req, err := http.NewRequest(
		http.MethodPost,
		strings.TrimRight(endpoint, "/")+"/api/login/oauth/access_token",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("build casdoor token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("casdoor token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read casdoor token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("casdoor token request returned %d: %s", resp.StatusCode, string(body))
	}

	var parsed casdoorTokenResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parse casdoor token response: %w", err)
	}

	if parsed.AccessToken == "" {
		return nil, fmt.Errorf("casdoor token response missing access_token")
	}

	// Refresh slightly early to avoid a request racing right against expiry.
	expiresAt := time.Now().Add(time.Duration(parsed.ExpiresIn)*time.Second - 30*time.Second)

	return &cachedToken{
		accessToken: parsed.AccessToken,
		expiresAt:   expiresAt,
	}, nil
}
