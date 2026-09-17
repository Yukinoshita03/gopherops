package service

import (
	"context"
	"errors"
	"strings"

	"github.com/Yukinoshita03/gopherops/internal/identity/application/port"
	"github.com/Yukinoshita03/gopherops/internal/identity/application/usecase"
	"github.com/Yukinoshita03/gopherops/internal/identity/domain"
	"golang.org/x/crypto/bcrypt"
)

type LoginService struct {
	userRepo port.UserRepository
}

func NewLoginService(userRepo port.UserRepository) *LoginService {
	return &LoginService{userRepo: userRepo}
}

var _ usecase.LoginUseCase = (*LoginService)(nil)

func (s *LoginService) Execute(
	ctx context.Context,
	input usecase.LoginInput,
) (usecase.LoginOutput, error) {
	username := strings.TrimSpace(input.Username)
	if username == "" || input.Password == "" {
		return usecase.LoginOutput{}, usecase.ErrInvalidLoginInput
	}

	user, err := s.userRepo.GetByUsername(ctx, username)
	if errors.Is(err, domain.ErrUserNotFound) {
		return usecase.LoginOutput{}, usecase.ErrInvalidCredentials
	}
	if err != nil {
		return usecase.LoginOutput{}, err
	}
	if user == nil {
		return usecase.LoginOutput{}, usecase.ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(input.Password),
	); errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return usecase.LoginOutput{}, usecase.ErrInvalidCredentials
	} else if err != nil {
		return usecase.LoginOutput{}, err
	}

	return usecase.LoginOutput{UserID: user.ID}, nil
}
