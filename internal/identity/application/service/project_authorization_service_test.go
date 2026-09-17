package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Yukinoshita03/gopherops/internal/identity/application/port"
	"github.com/Yukinoshita03/gopherops/internal/identity/application/service"
	"github.com/Yukinoshita03/gopherops/internal/identity/application/usecase"
	"github.com/Yukinoshita03/gopherops/internal/identity/domain"
)

type fakeProjectMembershipRepository struct {
	calls     int
	ctx       context.Context
	userID    int64
	projectID int64
	isMember  bool
	err       error
}

func (f *fakeProjectMembershipRepository) IsProjectMember(
	ctx context.Context,
	userID, projectID int64,
) (bool, error) {
	f.calls++
	f.ctx = ctx
	f.userID = userID
	f.projectID = projectID
	return f.isMember, f.err
}

var _ port.ProjectMembershipRepository = (*fakeProjectMembershipRepository)(nil)

func TestProjectAuthorizationServiceAllowsProjectMember(t *testing.T) {
	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "request marker")
	repo := &fakeProjectMembershipRepository{isMember: true}
	svc := service.NewProjectAuthorizationService(repo)

	err := svc.Execute(ctx, usecase.AuthorizeProjectInput{UserID: 17, ProjectID: 9})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if repo.calls != 1 || repo.userID != 17 || repo.projectID != 9 {
		t.Fatalf("membership lookup = calls:%d user:%d project:%d, want calls:1 user:17 project:9",
			repo.calls, repo.userID, repo.projectID)
	}
	if repo.ctx != ctx || repo.ctx.Value(contextKey{}) != "request marker" {
		t.Fatal("Execute() did not pass the request context to the repository")
	}
}

func TestProjectAuthorizationServiceDeniesNonMember(t *testing.T) {
	repo := &fakeProjectMembershipRepository{isMember: false}
	svc := service.NewProjectAuthorizationService(repo)

	err := svc.Execute(context.Background(), usecase.AuthorizeProjectInput{UserID: 17, ProjectID: 9})
	if !errors.Is(err, domain.ErrProjectAccessDenied) {
		t.Fatalf("Execute() error = %v, want ErrProjectAccessDenied", err)
	}
	if repo.calls != 1 {
		t.Fatalf("membership lookups = %d, want 1", repo.calls)
	}
}

func TestProjectAuthorizationServicePropagatesRepositoryError(t *testing.T) {
	wantErr := errors.New("membership storage unavailable")
	repo := &fakeProjectMembershipRepository{err: wantErr}
	svc := service.NewProjectAuthorizationService(repo)

	err := svc.Execute(context.Background(), usecase.AuthorizeProjectInput{UserID: 17, ProjectID: 9})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want wrapped %v", err, wantErr)
	}
}

func TestProjectAuthorizationServiceRejectsInvalidIDs(t *testing.T) {
	tests := []struct {
		name  string
		input usecase.AuthorizeProjectInput
	}{
		{name: "zero user ID", input: usecase.AuthorizeProjectInput{UserID: 0, ProjectID: 9}},
		{name: "zero project ID", input: usecase.AuthorizeProjectInput{UserID: 17, ProjectID: 0}},
		{name: "negative user ID", input: usecase.AuthorizeProjectInput{UserID: -1, ProjectID: 9}},
		{name: "negative project ID", input: usecase.AuthorizeProjectInput{UserID: 17, ProjectID: -1}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := &fakeProjectMembershipRepository{}
			svc := service.NewProjectAuthorizationService(repo)

			err := svc.Execute(context.Background(), test.input)
			if !errors.Is(err, usecase.ErrInvalidProjectAuthorizationInput) {
				t.Fatalf("Execute() error = %v, want ErrInvalidProjectAuthorizationInput", err)
			}
			if repo.calls != 0 {
				t.Fatalf("membership lookups = %d, want 0 for invalid IDs", repo.calls)
			}
		})
	}
}
