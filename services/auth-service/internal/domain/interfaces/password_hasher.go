package interfaces

import "context"

type PasswordHasher interface {
	Hash(ctx context.Context, plain string) (string, error)
	Verify(ctx context.Context, encoded, plain string) error
}
