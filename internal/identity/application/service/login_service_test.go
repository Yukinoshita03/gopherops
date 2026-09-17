package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Yukinoshita03/gopherops/internal/identity/application/service"
	"github.com/Yukinoshita03/gopherops/internal/identity/application/usecase"
	"github.com/Yukinoshita03/gopherops/internal/identity/domain"
	"golang.org/x/crypto/bcrypt"
)

func TestLoginServiceExecute(t *testing.T) {
	password := "correct-password"
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	repo := &fakeUserRepository{
		userByUsername: &domain.User{
			ID:           42,
			Username:     "alice",
			PasswordHash: string(passwordHash),
		},
	}
	loginService := service.NewLoginService(repo)

	output, err := loginService.Execute(context.Background(), usecase.LoginInput{
		Username: " alice ",
		Password: password,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if output.UserID != 42 {
		t.Fatalf("UserID = %d, want 42", output.UserID)
	}
	if repo.getByUsernameCalls != 1 {
		t.Fatalf("GetByUsername() calls = %d, want 1", repo.getByUsernameCalls)
	}
	if repo.queriedUsername != "alice" {
		t.Fatalf("queried username = %q, want %q", repo.queriedUsername, "alice")
	}
}

func TestLoginServiceExecuteReturnsSameErrorForUnknownUserAndWrongPassword(t *testing.T) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	tests := []struct {
		name string
		repo *fakeUserRepository
	}{
		{
			name: "unknown user",
			repo: &fakeUserRepository{getByUsernameErr: domain.ErrUserNotFound},
		},
		{
			name: "wrong password",
			repo: &fakeUserRepository{userByUsername: &domain.User{
				ID:           42,
				Username:     "alice",
				PasswordHash: string(passwordHash),
			}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			loginService := service.NewLoginService(test.repo)
			_, err := loginService.Execute(context.Background(), usecase.LoginInput{
				Username: "alice",
				Password: "wrong-password",
			})
			if !errors.Is(err, usecase.ErrInvalidCredentials) {
				t.Fatalf("error = %v, want ErrInvalidCredentials", err)
			}
		})
	}
}

func TestLoginServiceExecuteRejectsInvalidInputWithoutRepositoryCall(t *testing.T) {
	tests := []struct {
		name  string
		input usecase.LoginInput
	}{
		{name: "empty username", input: usecase.LoginInput{Password: "secret"}},
		{name: "whitespace username", input: usecase.LoginInput{Username: "   ", Password: "secret"}},
		{name: "empty password", input: usecase.LoginInput{Username: "alice"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := &fakeUserRepository{}
			loginService := service.NewLoginService(repo)
			_, err := loginService.Execute(context.Background(), test.input)
			if !errors.Is(err, usecase.ErrInvalidLoginInput) {
				t.Fatalf("error = %v, want ErrInvalidLoginInput", err)
			}
			if repo.getByUsernameCalls != 0 {
				t.Fatalf("GetByUsername() calls = %d, want 0", repo.getByUsernameCalls)
			}
		})
	}
}

func TestLoginServiceExecutePropagatesRepositoryError(t *testing.T) {
	wantErr := errors.New("database unavailable")
	repo := &fakeUserRepository{getByUsernameErr: wantErr}
	loginService := service.NewLoginService(repo)

	_, err := loginService.Execute(context.Background(), usecase.LoginInput{
		Username: "alice",
		Password: "secret",
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
}

func TestLoginServiceExecutePropagatesMalformedStoredHash(t *testing.T) {
	repo := &fakeUserRepository{userByUsername: &domain.User{
		ID:           42,
		Username:     "alice",
		PasswordHash: "not-a-bcrypt-hash",
	}}
	loginService := service.NewLoginService(repo)

	_, err := loginService.Execute(context.Background(), usecase.LoginInput{
		Username: "alice",
		Password: "secret",
	})
	if err == nil || errors.Is(err, usecase.ErrInvalidCredentials) {
		t.Fatalf("error = %v, want stored hash error", err)
	}
}
