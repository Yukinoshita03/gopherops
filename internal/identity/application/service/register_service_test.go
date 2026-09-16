package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Yukinoshita03/gopherops/internal/identity/application/port"
	"github.com/Yukinoshita03/gopherops/internal/identity/application/service"
	"github.com/Yukinoshita03/gopherops/internal/identity/application/usecase"
	"github.com/Yukinoshita03/gopherops/internal/identity/domain"
	"golang.org/x/crypto/bcrypt"
)

var _ port.UserRepository = (*fakeUserRepository)(nil)

type fakeUserRepository struct {
	createCalls int
	createdUser *domain.User
	nextID      int64
	createErr   error
}

func (f *fakeUserRepository) Create(
	_ context.Context,
	user *domain.User,
) error {
	f.createCalls++
	if f.createErr != nil {
		return f.createErr
	}

	copyOfUser := *user
	f.createdUser = &copyOfUser
	user.ID = f.nextID
	user.CreatedAt = time.Date(2026, time.September, 16, 0, 0, 0, 0, time.UTC)
	return nil
}

func (f *fakeUserRepository) GetByUsername(
	context.Context,
	string,
) (*domain.User, error) {
	return nil, nil
}

func (f *fakeUserRepository) GetByID(
	context.Context,
	int64,
) (*domain.User, error) {
	return nil, nil
}

func TestRegisterServiceExecute(t *testing.T) {
	repo := &fakeUserRepository{nextID: 42}
	registerService := service.NewRegisterService(repo)

	output, err := registerService.Execute(context.Background(), usecase.RegisterInput{
		Username: " alice ",
		Password: "plain-password",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if output.UserID != 42 {
		t.Fatalf("UserID = %d, want 42", output.UserID)
	}
	if repo.createCalls != 1 {
		t.Fatalf("Create() calls = %d, want 1", repo.createCalls)
	}
	if repo.createdUser == nil {
		t.Fatal("Create() did not receive a user")
	}
	if repo.createdUser.Username != "alice" {
		t.Errorf("Username = %q, want %q", repo.createdUser.Username, "alice")
	}
	if repo.createdUser.PasswordHash == "plain-password" {
		t.Fatal("PasswordHash contains the plaintext password")
	}
	if err := bcrypt.CompareHashAndPassword(
		[]byte(repo.createdUser.PasswordHash),
		[]byte("plain-password"),
	); err != nil {
		t.Fatalf("PasswordHash does not match the password: %v", err)
	}
}

func TestRegisterServiceExecuteRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input usecase.RegisterInput
	}{
		{
			name: "empty username",
			input: usecase.RegisterInput{
				Password: "plain-password",
			},
		},
		{
			name: "whitespace username",
			input: usecase.RegisterInput{
				Username: "   ",
				Password: "plain-password",
			},
		},
		{
			name: "empty password",
			input: usecase.RegisterInput{
				Username: "alice",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := &fakeUserRepository{nextID: 42}
			registerService := service.NewRegisterService(repo)

			_, err := registerService.Execute(context.Background(), test.input)
			if !errors.Is(err, usecase.ErrInvalidRegisterInput) {
				t.Fatalf("error = %v, want ErrInvalidRegisterInput", err)
			}
			if repo.createCalls != 0 {
				t.Fatalf("Create() calls = %d, want 0", repo.createCalls)
			}
		})
	}
}

func TestRegisterServiceExecutePropagatesRepositoryError(t *testing.T) {
	wantErr := errors.New("database unavailable")
	repo := &fakeUserRepository{createErr: wantErr}
	registerService := service.NewRegisterService(repo)

	_, err := registerService.Execute(context.Background(), usecase.RegisterInput{
		Username: "alice",
		Password: "plain-password",
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
	if repo.createCalls != 1 {
		t.Fatalf("Create() calls = %d, want 1", repo.createCalls)
	}
}
