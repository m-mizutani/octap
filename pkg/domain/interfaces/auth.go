package interfaces

import "context"

type AuthService interface {
	GetToken(ctx context.Context) (string, error)
	SaveToken(ctx context.Context, token string) error
	DeviceFlow(ctx context.Context) (string, error)
}
