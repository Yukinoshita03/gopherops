package usecase

import (
	"context"
	"errors"
)

var ErrInvalidProjectAuthorizationInput = errors.New("invalid project authorization input")

// AuthorizeProjectInput 用可信的当前用户 ID 检查其项目访问权。
// UserID 应来自已验证的身份上下文，不能从请求体中读取。
type AuthorizeProjectInput struct {
	UserID    int64
	ProjectID int64
}

// AuthorizeProjectUseCase 在用户没有项目成员关系时返回 domain.ErrProjectAccessDenied。
type AuthorizeProjectUseCase interface {
	Execute(ctx context.Context, input AuthorizeProjectInput) error
}
