package port

import (
	"context"

	"github.com/Yukinoshita03/gopherops/internal/identity/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
	GetByID(ctx context.Context, id int64) (*domain.User, error)
}
