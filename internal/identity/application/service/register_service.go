package service

import (
	"context"
	"strings"

	"github.com/Yukinoshita03/gopherops/internal/identity/application/port"
	"github.com/Yukinoshita03/gopherops/internal/identity/application/usecase"
	"github.com/Yukinoshita03/gopherops/internal/identity/domain"
	"golang.org/x/crypto/bcrypt"
)

type RegisterService struct {
	userRepo port.UserRepository
}

func NewRegisterService(userRepo port.UserRepository) *RegisterService {
	return &RegisterService{
		userRepo: userRepo,
	}
}

var _ usecase.RegisterUseCase = (*RegisterService)(nil)

func (s *RegisterService) Execute(
	ctx context.Context,
	input usecase.RegisterInput,
) (usecase.RegisterOutput, error) {
	username := strings.TrimSpace(input.Username)
	if username == "" || input.Password == "" {
		return usecase.RegisterOutput{}, usecase.ErrInvalidRegisterInput
	}

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(input.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return usecase.RegisterOutput{}, err
	}

	user := &domain.User{
		Username:     username,
		PasswordHash: string(passwordHash),
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return usecase.RegisterOutput{}, err
	}

	return usecase.RegisterOutput{UserID: user.ID}, nil
}
