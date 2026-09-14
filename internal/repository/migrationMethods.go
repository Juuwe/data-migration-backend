package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Juuwe/data-migration-backend/internal/ds"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	ErrNotFound = ds.ErrMigrationMethodNotFound
)

type PosrtgresMigrationMethodsRepo struct {
	db    *gorm.DB
	rawDB *sql.DB
}

type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
	Timezone string
}

func New(cfg Config) (*PosrtgresMigrationMethodsRepo, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode, cfg.Timezone)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	rawDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get raw database connection: %w", err)
	}

	return &PosrtgresMigrationMethodsRepo{
		db:    db,
		rawDB: rawDB,
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
			return ds.MigrationMethod{}, ErrNotFound
		}
		return ds.MigrationMethod{}, fmt.Errorf("db error: %w", err)
	}

	return method, nil
}

func (r *PosrtgresMigrationMethodsRepo) FindDraft(ctx context.Context, creatorID int64) (ds.MigrationMethod, error) {
	var method ds.MigrationMethod

	err := r.db.WithContext(ctx).
		Where("creator_id = ? AND status = ?", creatorID, ds.StatusDraft).
		First(&method).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ds.MigrationMethod{}, ErrNotFound
		}
		return ds.MigrationMethod{}, fmt.Errorf("db error: %w", err)
	}

	return method, nil
}

func (r *PosrtgresMigrationMethodsRepo) FindPublished(ctx context.Context) ([]ds.MigrationMethod, error) {
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

func (r *PosrtgresMigrationMethodsRepo) FindPublishedByTime(ctx context.Context, ltime, rtime float64) ([]ds.MigrationMethod, error) {
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

func (r *PosrtgresMigrationMethodsRepo) CountMethodLikesByID(ctx context.Context, methodID int64) (int, error) {
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

func (r *PosrtgresMigrationMethodsRepo) CountLikesByMethodIDs(ctx context.Context, methodIDs []int64) (map[int64]int, error) {
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

func (r *PosrtgresMigrationMethodsRepo) Create(ctx context.Context, m *ds.MigrationMethod) error {
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("create migration method: %w", err)
	}
	return nil
}

func (r *PosrtgresMigrationMethodsRepo) Update(ctx context.Context, m *ds.MigrationMethod) error {
	result := r.db.WithContext(ctx).Model(m).Updates(map[string]any{
		"description": m.Description,
		"time_in_gb":  m.TimeInGb,
		"reliability": m.Reliability,
		"status":      m.Status,
		"formed_at":   m.FormedAt,
	})
	if result.Error != nil {
		return fmt.Errorf("update migration method: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PosrtgresMigrationMethodsRepo) SoftDeleteSQL(ctx context.Context, id int64) error {
	query := `UPDATE migration_methods SET status = 'deleted' WHERE id = $1 AND status != 'deleted'`

	result, err := r.rawDB.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete migration method: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("%w: карточка не найдена или уже удалена", ErrNotFound)
	}

	return nil
}
