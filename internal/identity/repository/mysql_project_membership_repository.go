package repository

import (
	"context"

	"github.com/Yukinoshita03/gopherops/internal/identity/application/port"
	"gorm.io/gorm"
)

// MySQLProjectMembershipRepository checks project membership in MySQL.
type MySQLProjectMembershipRepository struct {
	db *gorm.DB
}

func NewMySQLProjectMembershipRepository(db *gorm.DB) *MySQLProjectMembershipRepository {
	return &MySQLProjectMembershipRepository{db: db}
}

var _ port.ProjectMembershipRepository = (*MySQLProjectMembershipRepository)(nil)

func (r *MySQLProjectMembershipRepository) IsProjectMember(
	ctx context.Context,
	userID, projectID int64,
) (bool, error) {
	var exists int64
	result := r.db.WithContext(ctx).Raw(
		`SELECT EXISTS (
			SELECT 1
			FROM project_members
			WHERE project_id = ? AND user_id = ?
		)`,
		projectID,
		userID,
	).Scan(&exists)
	if result.Error != nil {
		return false, result.Error
	}

	return exists != 0, nil
}
