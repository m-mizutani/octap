package interfaces

import (
	"context"

	"github.com/google/go-github/v74/github"
)

type AuthService interface {
	GetToken(ctx context.Context) (string, error)
	SaveToken(ctx context.Context, token string) error
	DeviceFlow(ctx context.Context) (string, error)
	GetAuthenticatedClient(ctx context.Context) (*github.Client, error)
}
