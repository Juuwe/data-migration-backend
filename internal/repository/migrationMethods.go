package repository

import (
	"fmt"
	"errors"
	"context"

	"github.com/Juuwe/data-migration-backend/internal/ds"

	"gorm.io/gorm"
	"gorm.io/driver/postgres"
)

var (
	ErrNotFound = errors.New("method not found")
)

type PosrtgresMigrationMethodsRepo struct {
	db *gorm.DB
}

type Config struct {
	Host string
	Port int
	User string
	Password string
	DBName string
	SSLMode string
	Timezone string
}

func New(cfg Config) (*PosrtgresMigrationMethodsRepo, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &PosrtgresMigrationMethodsRepo{
		db: db,
	}, nil
}

func (r *PosrtgresMigrationMethodsRepo) FindByID(ctx context.Context, id int64) (ds.MigrationMethod, error) {
	var method ds.MigrationMethod

	err := r.db.WithContext(ctx).First(&method, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ds.MigrationMethod{}, ErrNotFound
		}
		return ds.MigrationMethod{}, fmt.Errorf("db error: %w", err)
	}

	return method, nil
}

func (r *PosrtgresMigrationMethodsRepo) FindNextPublishedAfterID(ctx context.Context, id int64) (ds.MigrationMethod, error) {
	var method ds.MigrationMethod

	err := r.db.WithContext(ctx).
		Where("id > ? AND status = ?", id, ds.StatusPublished).
		Order("id ASC").
		First(&method).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ds.MigrationMethod{}, errors.New("это последняя опубликованная миграция")
		}
		return ds.MigrationMethod{}, fmt.Errorf("db error: %w", err)
	}

	return method, nil
}

func (r *PosrtgresMigrationMethodsRepo) FindDraft(ctx context.Context) (ds.MigrationMethod, error) {
	var method ds.MigrationMethod

	err := r.db.WithContext(ctx).
		Where("status = ?", ds.StatusDraft).
		First(&method).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ds.MigrationMethod{}, errors.New("черновиков не найдено")
		}
		return ds.MigrationMethod{}, fmt.Errorf("db error: %w", err)
	}

	return method, nil
}

func (r *PosrtgresMigrationMethodsRepo) FindPublished(ctx context.Context) ([]ds.MigrationMethod, error) {
	var methods []ds.MigrationMethod

	err := r.db.WithContext(ctx).
		Where("status = ?", ds.StatusPublished).
		Find(&methods).Error

	if err != nil {
		return nil, fmt.Errorf("db error: %w", err)
	}
	if len(methods) == 0 {
		return nil, errors.New("опубликованных миграций не найдено")
	}

	return methods, nil
}

func (r *PosrtgresMigrationMethodsRepo) FindPublishedByTime(ctx context.Context, ltime, rtime float64) ([]ds.MigrationMethod, error) {
	var methods []ds.MigrationMethod

	err := r.db.WithContext(ctx).
		Where("status = ? AND time_in_gb BETWEEN ? AND ?", ds.StatusPublished, ltime, rtime).
		Find(&methods).Error

	if err != nil {
		return nil, fmt.Errorf("db error: %w", err)
	}
	if len(methods) == 0 {
		return nil, errors.New("отфильтрованных миграций не найдено")
	}

	return methods, nil
}

func (r *PosrtgresMigrationMethodsRepo) CountMethodLikesByID(ctx context.Context, methodID int) (int64, error) {
	var count int64

	err := r.db.WithContext(ctx).Table("likes").Where("method_id = ?", methodID).Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("db error counting likes: %w", err)
	}

	return count, nil
}
