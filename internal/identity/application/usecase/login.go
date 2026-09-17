package usecase

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInvalidLoginInput  = errors.New("invalid login input")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type LoginInput struct {
	Username string
	Password string
}

type LoginOutput struct {
	UserID      int64
	AccessToken string
	ExpiresAt   time.Time
}

type LoginUseCase interface {
	Execute(ctx context.Context, input LoginInput) (LoginOutput, error)
}
