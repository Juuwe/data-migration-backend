package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/Juuwe/data-migration-backend/internal/ds"
	"gorm.io/gorm"
)

func (r *PostgresRepository) CreateUser(ctx context.Context, user *ds.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ds.ErrUserAlreadyExists
		}
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *PostgresRepository) FindUserByEmail(ctx context.Context, email string) (ds.User, error) {
	var user ds.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ds.User{}, ds.ErrUserNotFound
		}
		return ds.User{}, fmt.Errorf("find user by email: %w", err)
	}
	return user, nil
}

func (r *PostgresRepository) FindUserByID(ctx context.Context, id int64) (ds.User, error) {
	var user ds.User
	if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ds.User{}, ds.ErrUserNotFound
		}
		return ds.User{}, fmt.Errorf("find user by ID: %w", err)
	}
	return user, nil
}
