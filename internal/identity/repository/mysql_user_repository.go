package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Yukinoshita03/gopherops/internal/identity/application/port"
	"github.com/Yukinoshita03/gopherops/internal/identity/domain"
	mysqldriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

type userModel struct {
	ID           int64     `gorm:"column:id;primaryKey"`
	Username     string    `gorm:"column:username"`
	PasswordHash string    `gorm:"column:password_hash"`
	CreatedAt    time.Time `gorm:"column:created_at"`
}

func (userModel) TableName() string {
	return "users"
}

type MySQLUserRepository struct {
	db *gorm.DB
}

func NewMySQLUserRepository(db *gorm.DB) *MySQLUserRepository {
	return &MySQLUserRepository{db: db}
}

var _ port.UserRepository = (*MySQLUserRepository)(nil)

func (r *MySQLUserRepository) Create(
	ctx context.Context,
	user *domain.User,
) error {
	var model userModel
	model.Username = user.Username
	model.PasswordHash = user.PasswordHash
	db := r.db.WithContext(ctx).Create(&model)
	if db.Error != nil {
		if isUsernameUniqueConstraintError(db.Error) {
			return domain.ErrUserAlreadyExists
		}
		return db.Error
	}
	user.ID = model.ID
	user.CreatedAt = model.CreatedAt
	return nil
}

func isUsernameUniqueConstraintError(err error) bool {
	var mysqlErr *mysqldriver.MySQLError
	return errors.As(err, &mysqlErr) &&
		mysqlErr.Number == 1062 &&
		strings.Contains(mysqlErr.Message, "uk_users_username")
}

func (r *MySQLUserRepository) GetByUsername(
	ctx context.Context,
	username string,
) (*domain.User, error) {
	var model userModel
	model.Username = username
	db := r.db.WithContext(ctx).Where("username = ?", username).First(&model)
	var domainUser = toDomainUser(model)
	if errors.Is(db.Error, gorm.ErrRecordNotFound) {
		return nil, domain.ErrUserNotFound
	}
	if db.Error != nil {
		return nil, db.Error
	}
	return domainUser, nil
}

func (r *MySQLUserRepository) GetByID(
	ctx context.Context,
	id int64,
) (*domain.User, error) {
	var model userModel
	model.ID = id
	db := r.db.WithContext(ctx).Where("id = ?", id).First(&model)
	var domainUser = toDomainUser(model)
	if errors.Is(db.Error, gorm.ErrRecordNotFound) {
		return nil, domain.ErrUserNotFound
	}
	if db.Error != nil {
		return nil, db.Error
	}

	return domainUser, nil
}

func toDomainUser(model userModel) *domain.User {
	return &domain.User{
		ID:           model.ID,
		Username:     model.Username,
		PasswordHash: model.PasswordHash,
		CreatedAt:    model.CreatedAt,
	}
}
