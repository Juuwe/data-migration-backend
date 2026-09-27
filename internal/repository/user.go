package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/Juuwe/data-migration-backend/internal/ds"
	"gorm.io/gorm"
)

func (r *PosrtgresMigrationMethodsRepo) CreateUser(ctx context.Context, user *ds.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ds.ErrUserAlreadyExists
		}
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}
