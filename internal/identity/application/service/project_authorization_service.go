package service

import (
	"context"
	"fmt"

	"github.com/Yukinoshita03/gopherops/internal/identity/application/port"
	"github.com/Yukinoshita03/gopherops/internal/identity/application/usecase"
	"github.com/Yukinoshita03/gopherops/internal/identity/domain"
)

type ProjectAuthorizationService struct {
	membershipRepo port.ProjectMembershipRepository
}

func NewProjectAuthorizationService(
	membershipRepo port.ProjectMembershipRepository,
) *ProjectAuthorizationService {
	return &ProjectAuthorizationService{membershipRepo: membershipRepo}
}

var _ usecase.AuthorizeProjectUseCase = (*ProjectAuthorizationService)(nil)

func (s *ProjectAuthorizationService) Execute(
	ctx context.Context,
	input usecase.AuthorizeProjectInput,
) error {
	if input.UserID <= 0 || input.ProjectID <= 0 {
		return usecase.ErrInvalidProjectAuthorizationInput
	}

	isMember, err := s.membershipRepo.IsProjectMember(ctx, input.UserID, input.ProjectID)
	if err != nil {
		return fmt.Errorf("check project membership: %w", err)
	}
	if !isMember {
		return domain.ErrProjectAccessDenied
	}
	return nil
}
