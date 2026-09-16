package usecase

import (
	"context"
	"errors"
)

var ErrInvalidRegisterInput = errors.New("invalid register input")

type RegisterInput struct {
	Username string
	Password string
}

type RegisterOutput struct {
	UserID int64
}

type RegisterUseCase interface {
	Execute(ctx context.Context, input RegisterInput) (RegisterOutput, error)
}
