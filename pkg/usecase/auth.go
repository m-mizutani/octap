package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/go-github/v74/github"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/octap/pkg/domain"
	"github.com/m-mizutani/octap/pkg/domain/interfaces"
	"golang.org/x/oauth2"
)

type AuthService struct {
	storage *TokenStorage
	logger  *slog.Logger
}

func NewAuthService(logger *slog.Logger) interfaces.AuthService {
	return &AuthService{
		storage: NewTokenStorage(),
		logger:  logger,
	}
}

func (s *AuthService) GetToken(ctx context.Context) (string, error) {
	return s.storage.GetToken(ctx)
}

func (s *AuthService) SaveToken(ctx context.Context, token string) error {
	return s.storage.SaveToken(ctx, token)
}

func (s *AuthService) DeviceFlow(ctx context.Context) (string, error) {
	// Note: GitHub Device Flow is not yet available in go-github v74
	// For now, we'll use a personal access token approach
	// TODO: Implement device flow when available or use direct API calls

	fmt.Printf("\n")
	fmt.Printf("🔐 GitHub Authentication Required\n")
	fmt.Printf("────────────────────────────────────\n")
	fmt.Printf("Please create a personal access token:\n")
	fmt.Printf("1. Visit: https://github.com/settings/tokens/new\n")
	fmt.Printf("2. Select 'repo' scope\n")
	fmt.Printf("3. Generate token\n")
	fmt.Printf("4. Enter token: ")

	var token string
	_, _ = fmt.Scanln(&token)

	if token == "" {
		return "", domain.ErrAuthentication.Wrap(goerr.New("no token provided"))
	}

	// Validate token
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	_, _, err := client.Users.Get(ctx, "")
	if err != nil {
		return "", domain.ErrAuthentication.Wrap(err)
	}

	if err := s.SaveToken(ctx, token); err != nil {
		return "", err
	}

	fmt.Printf("✅ Authentication successful!\n\n")
	return token, nil
}

func (s *AuthService) GetAuthenticatedClient(ctx context.Context) (*github.Client, error) {
	token, err := s.GetToken(ctx)
	if err != nil {
		return nil, err
	}

	if token == "" {
		s.logger.Debug("No saved token found, starting authentication")
		token, err = s.DeviceFlow(ctx)
		if err != nil {
			return nil, err
		}
	}

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc), nil
}
