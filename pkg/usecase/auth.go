package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/go-github/v74/github"
	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/octap/pkg/domain"
	"github.com/m-mizutani/octap/pkg/domain/interfaces"
	"golang.org/x/oauth2"
)

const (
	// GitHub Device Flow endpoints
	deviceCodeURL = "https://github.com/login/device/code"
	tokenURL      = "https://github.com/login/oauth/access_token" // #nosec G101 - This is not a credential, it's a public API endpoint
)

type deviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	Error       string `json:"error"`
	ErrorDesc   string `json:"error_description"`
}

type AuthService struct {
	storage  *TokenStorage
	clientID string
}

func NewAuthService(clientID string) interfaces.AuthService {
	// Use default client ID if not provided
	if clientID == "" {
		clientID = "Ov23litxvfoH9DYHtwKP"
	}

	return &AuthService{
		storage:  NewTokenStorage(),
		clientID: clientID,
	}
}

func (s *AuthService) GetToken(ctx context.Context) (string, error) {
	return s.storage.GetToken(ctx)
}

func (s *AuthService) SaveToken(ctx context.Context, token string) error {
	return s.storage.SaveToken(ctx, token)
}

func (s *AuthService) DeviceFlow(ctx context.Context) (string, error) {
	// GitHub Device Flow implementation using direct API calls
	deviceCode, err := s.requestDeviceCode(ctx)
	if err != nil {
		return "", err
	}

	fmt.Printf("\n")
	fmt.Printf("🔐 GitHub Device Flow Authentication\n")
	fmt.Printf("────────────────────────────────────\n")
	fmt.Printf("1. Copy this code: %s\n", deviceCode.UserCode)
	fmt.Printf("2. Visit: %s\n", deviceCode.VerificationURI)
	fmt.Printf("3. Paste the code and authorize the app\n")
	fmt.Printf("\n")
	fmt.Printf("⏳ Waiting for authorization...\n")

	token, err := s.pollForToken(ctx, deviceCode)
	if err != nil {
		return "", err
	}

	if err := s.SaveToken(ctx, token); err != nil {
		return "", err
	}

	fmt.Printf("✅ Authentication successful!\n\n")
	return token, nil
}

func (s *AuthService) GetAuthenticatedClient(ctx context.Context) (*github.Client, error) {
	logger := ctxlog.From(ctx)
	token, err := s.GetToken(ctx)
	if err != nil {
		return nil, err
	}

	if token == "" {
		logger.Debug("No saved token found, starting authentication")
		token, err = s.DeviceFlow(ctx)
		if err != nil {
			return nil, err
		}
	}

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc), nil
}

func (s *AuthService) requestDeviceCode(ctx context.Context) (*deviceCodeResponse, error) {
	reqBody := bytes.NewBufferString(fmt.Sprintf("client_id=%s&scope=repo", s.clientID))

	req, err := http.NewRequestWithContext(ctx, "POST", deviceCodeURL, reqBody)
	if err != nil {
		return nil, domain.ErrAuthentication.Wrap(err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, domain.ErrAuthentication.Wrap(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, domain.ErrAuthentication.Wrap(err)
	}

	var deviceCode deviceCodeResponse
	if err := json.Unmarshal(body, &deviceCode); err != nil {
		return nil, domain.ErrAuthentication.Wrap(err)
	}

	if deviceCode.DeviceCode == "" {
		return nil, domain.ErrAuthentication.Wrap(goerr.New("failed to get device code - check your Client ID"))
	}

	return &deviceCode, nil
}

func (s *AuthService) pollForToken(ctx context.Context, deviceCode *deviceCodeResponse) (string, error) {
	interval := time.Duration(deviceCode.Interval) * time.Second
	if interval == 0 {
		interval = 5 * time.Second
	}

	deadline := time.Now().Add(time.Duration(deviceCode.ExpiresIn) * time.Second)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	client := &http.Client{Timeout: 30 * time.Second}

	for {
		select {
		case <-ctx.Done():
			return "", domain.ErrAuthentication.Wrap(ctx.Err())
		case <-ticker.C:
			if time.Now().After(deadline) {
				return "", domain.ErrAuthentication.Wrap(goerr.New("device code expired"))
			}

			reqBody := bytes.NewBufferString(fmt.Sprintf(
				"client_id=%s&device_code=%s&grant_type=urn:ietf:params:oauth:grant-type:device_code",
				s.clientID, deviceCode.DeviceCode,
			))

			req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, reqBody)
			if err != nil {
				return "", domain.ErrAuthentication.Wrap(err)
			}
			req.Header.Set("Accept", "application/json")
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			resp, err := client.Do(req)
			if err != nil {
				logger := ctxlog.From(ctx)
				logger.Debug("error polling for token", slog.String("error", err.Error()))
				continue
			}

			body, err := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if err != nil {
				logger := ctxlog.From(ctx)
				logger.Debug("error reading response", slog.String("error", err.Error()))
				continue
			}

			var tokenResp tokenResponse
			if err := json.Unmarshal(body, &tokenResp); err != nil {
				logger := ctxlog.From(ctx)
				logger.Debug("error parsing response", slog.String("error", err.Error()))
				continue
			}

			if tokenResp.Error != "" {
				if tokenResp.Error == "authorization_pending" {
					// Still waiting for user authorization
					continue
				}
				if tokenResp.Error == "slow_down" {
					// Increase interval
					interval = interval + 5*time.Second
					ticker.Reset(interval)
					continue
				}
				// Other errors
				return "", domain.ErrAuthentication.Wrap(goerr.New(tokenResp.ErrorDesc))
			}

			if tokenResp.AccessToken != "" {
				return tokenResp.AccessToken, nil
			}
		}
	}
}
