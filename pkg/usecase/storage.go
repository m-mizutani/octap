package usecase

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/m-mizutani/octap/pkg/domain"
)

type TokenStorage struct {
	configDir string
}

func NewTokenStorage() *TokenStorage {
	homeDir, _ := os.UserHomeDir()
	return &TokenStorage{
		configDir: filepath.Join(homeDir, ".config", "octap"),
	}
}

func (s *TokenStorage) getTokenPath() string {
	return filepath.Join(s.configDir, "token.json")
}

type tokenData struct {
	AccessToken string `json:"access_token"`
}

func (s *TokenStorage) SaveToken(ctx context.Context, token string) error {
	if err := os.MkdirAll(s.configDir, 0700); err != nil {
		return domain.ErrConfiguration.Wrap(err)
	}

	data := tokenData{AccessToken: token}
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return domain.ErrConfiguration.Wrap(err)
	}

	tokenPath := s.getTokenPath()
	if err := os.WriteFile(tokenPath, jsonData, 0600); err != nil {
		return domain.ErrConfiguration.Wrap(err)
	}

	return nil
}

func (s *TokenStorage) GetToken(ctx context.Context) (string, error) {
	tokenPath := s.getTokenPath()
	data, err := os.ReadFile(tokenPath) // #nosec G304 - tokenPath is constructed from a fixed directory path
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", domain.ErrConfiguration.Wrap(err)
	}

	var token tokenData
	if err := json.Unmarshal(data, &token); err != nil {
		return "", domain.ErrConfiguration.Wrap(err)
	}

	return token.AccessToken, nil
}
