package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/Juuwe/data-migration-backend/internal/ds"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PostgresRepository struct {
	db *gorm.DB
}

func New(dsn string) (*PostgresRepository, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	return &PostgresRepository{db: db}, nil
}

func (r *PostgresRepository) Close() error {
	db, err := r.db.DB()
	if err != nil {
		return err
	}
	return db.Close()
}

func (r *PostgresRepository) FindByID(ctx context.Context, id int64) (ds.MigrationMethod, error) {
	var method ds.MigrationMethod

	err := r.db.WithContext(ctx).First(&method, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ds.MigrationMethod{}, ds.ErrMigrationMethodNotFound
		}
		return ds.MigrationMethod{}, fmt.Errorf("db error: %w", err)
	}

	return method, nil
}

func (r *PostgresRepository) FindNextPublishedAfterID(ctx context.Context, id int64) (ds.MigrationMethod, error) {
	var method ds.MigrationMethod

	err := r.db.WithContext(ctx).
		Where("id > ? AND status = ?", id, ds.StatusPublished).
		Order("id ASC").
		First(&method).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ds.MigrationMethod{}, ds.ErrMigrationMethodNotFound
		}
		return ds.MigrationMethod{}, fmt.Errorf("db error: %w", err)
	}

	return method, nil
}

func (r *PostgresRepository) FindDraft(ctx context.Context, creatorID int64) (ds.MigrationMethod, error) {
	var method ds.MigrationMethod

	err := r.db.WithContext(ctx).
		Where("creator_id = ? AND status = ?", creatorID, ds.StatusDraft).
		First(&method).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ds.MigrationMethod{}, ds.ErrMigrationMethodNotFound
		}
		return ds.MigrationMethod{}, fmt.Errorf("db error: %w", err)
	}

	return method, nil
}

func (r *PostgresRepository) FindPublished(ctx context.Context) ([]ds.MigrationMethod, error) {
	var methods []ds.MigrationMethod

	err := r.db.WithContext(ctx).
		Where("status = ?", ds.StatusPublished).
		Order("id ASC").
		Find(&methods).Error

	if err != nil {
		return nil, fmt.Errorf("db error: %w", err)
	}
	return methods, nil
}

func (r *PostgresRepository) FindPublishedByTime(ctx context.Context, ltime, rtime float64) ([]ds.MigrationMethod, error) {
	var methods []ds.MigrationMethod

	err := r.db.WithContext(ctx).
		Where("status = ? AND time_in_gb BETWEEN ? AND ?", ds.StatusPublished, ltime, rtime).
		Order("id ASC").
		Find(&methods).Error

	if err != nil {
		return nil, fmt.Errorf("db error: %w", err)
	}
	return methods, nil
}

func (r *PostgresRepository) CountMethodLikesByID(ctx context.Context, methodID int64) (int, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&ds.MigrationMethodLike{}).
		Where("method_id = ?", methodID).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("db error counting likes: %w", err)
	}

	return int(count), nil
}

func (r *PostgresRepository) IsLiked(ctx context.Context, methodID, userID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&ds.MigrationMethodLike{}).
		Where("method_id = ? AND user_id = ?", methodID, userID).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("check migration method like: %w", err)
	}
	return count > 0, nil
}

func (r *PostgresRepository) CountLikesByMethodIDs(ctx context.Context, methodIDs []int64) (map[int64]int, error) {
	if len(methodIDs) == 0 {
		return make(map[int64]int), nil
	}

	var res []struct {
		MethodID int64
		Count    int
	}

	err := r.db.WithContext(ctx).
		Model(&ds.MigrationMethodLike{}).
		Select("method_id, COUNT(*) AS count").
		Where("method_id IN ?", methodIDs).
		Group("method_id").
		Scan(&res).Error

	if err != nil {
		return nil, err
	}

	likesMap := make(map[int64]int, len(res))
	for i := range res {
		likesMap[res[i].MethodID] = res[i].Count
	}

	return likesMap, nil
}

func (r *PostgresRepository) Create(ctx context.Context, m *ds.MigrationMethod) error {
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("create migration method: %w", err)
	}
	return nil
}

func (r *PostgresRepository) Update(ctx context.Context, m *ds.MigrationMethod) error {
	result := r.db.WithContext(ctx).Model(m).Where("status = ?", ds.StatusDraft).Updates(map[string]any{
		"description":  m.Description,
		"time_in_gb":   m.TimeInGb,
		"reliability":  m.Reliability,
		"status":       m.Status,
		"published_at": m.PublishedAt,
	})
	if result.Error != nil {
		return fmt.Errorf("update migration method: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ds.ErrMigrationMethodNotFound
	}
	return nil
}

func (r *PostgresRepository) SoftDelete(ctx context.Context, id, creatorID int64) error {
	result := r.db.WithContext(ctx).Model(&ds.MigrationMethod{}).
		Where("id = ? AND creator_id = ? AND status IN ?", id, creatorID, []ds.Status{ds.StatusDraft, ds.StatusPublished}).
		Update("status", ds.StatusDeleted)
	if result.Error != nil {
		return fmt.Errorf("delete migration method: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ds.ErrMigrationMethodNotFound
	}
	return nil
}

func (r *PostgresRepository) SetLike(ctx context.Context, methodID, userID int64, like int) error {
	if like == 1 {
		entry := ds.MigrationMethodLike{MethodID: methodID, UserID: userID}
		if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&entry).Error; err != nil {
			return fmt.Errorf("like migration method: %w", err)
		}
		return nil
	}

	if err := r.db.WithContext(ctx).
		Where("method_id = ? AND user_id = ?", methodID, userID).
		Delete(&ds.MigrationMethodLike{}).Error; err != nil {
		return fmt.Errorf("unlike migration method: %w", err)
	}
	return nil
}
