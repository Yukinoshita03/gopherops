package usecase

import (
	"context"
	"errors"
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
	UserID int64
}

type LoginUseCase interface {
	Execute(ctx context.Context, input LoginInput) (LoginOutput, error)
}
