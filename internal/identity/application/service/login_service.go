package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Yukinoshita03/gopherops/internal/identity/application/port"
	"github.com/Yukinoshita03/gopherops/internal/identity/application/usecase"
	"github.com/Yukinoshita03/gopherops/internal/identity/domain"
	identitytoken "github.com/Yukinoshita03/gopherops/internal/identity/token"
	"golang.org/x/crypto/bcrypt"
)

type LoginService struct {
	userRepo port.UserRepository
	signer   identitytoken.Signer
}

func NewLoginService(userRepo port.UserRepository, signer identitytoken.Signer) *LoginService {
	return &LoginService{userRepo: userRepo, signer: signer}
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

	if s.signer == nil {
		return usecase.LoginOutput{}, errors.New("login token signer is not configured")
	}
	accessToken, expiresAt, err := s.signer.Sign(strconv.FormatInt(user.ID, 10))
	if err != nil {
		return usecase.LoginOutput{}, fmt.Errorf("sign access token: %w", err)
	}

	return usecase.LoginOutput{
		UserID:      user.ID,
		AccessToken: accessToken,
		ExpiresAt:   expiresAt,
	}, nil
}
