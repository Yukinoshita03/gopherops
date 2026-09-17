package port

import "context"

// ProjectMembershipRepository 检查用户是否属于指定项目。
type ProjectMembershipRepository interface {
	IsProjectMember(ctx context.Context, userID, projectID int64) (bool, error)
}
