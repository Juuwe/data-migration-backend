package repository

import (
	"context"
	"errors"

	"github.com/Juuwe/data-migration-backend/internal/app/model"
)

type InMemoryMigrationMethodRepository struct {
	migrationMethods []model.MigrationMethod
}

func NewInMemoryMigrationRepository() *InMemoryMigrationMethodRepository {
	migrationsMethods := []model.MigrationMethod{
		{
			ID:          1,
			Name:        "Онлайн",
			Title:       "Онлайн миграция",
			Description: "Перенос корпоративных данных...",
			VideoKey:    "online.mp4",
			ImageKey: "online.jpeg",
			Status:      model.StatusPublished, // Исправлено
			TimeInGb:    0.5,
			Reliability: 0.95,
			Likes:       []int{101, 102, 105, 108},
		},
		{
			ID:          2,
			Name:        "Офлайн",
			Title:       "Офлайн миграция",
			Description: "Перенос корпоративных данных...",
			VideoKey:    "offline.mp4",
			ImageKey: "offline.jpeg",
			Status:      model.StatusPublished,
			TimeInGb:    0.3,
			Reliability: 0.98,
			Likes:       []int{101, 105, 108},
		},
		{
			ID:          3,
			Name:        "Репликация",
			Title:       "Репликация данных",
			Description: "Перенос данных...",
			VideoKey:    "replication.mp4",
			ImageKey: "replication.jpeg",
			Status:      model.StatusDraft,
			TimeInGb:    1,
			Reliability: 0.998,
		},
	}

	return &InMemoryMigrationMethodRepository{migrationMethods: migrationsMethods}
}

func (r *InMemoryMigrationMethodRepository) FindByID(ctx context.Context, id int) (model.MigrationMethod, error) {
	for _, m := range r.migrationMethods {
		if m.ID == id {
			return m, nil
		}
	}

	return model.MigrationMethod{}, errors.New("method not found")
}

func (r *InMemoryMigrationMethodRepository) FindNextPublishedAfterID(ctx context.Context, id int) (model.MigrationMethod, error) {
	currentIndex := -1
	for i, m := range r.migrationMethods {
		if m.ID == id {
			currentIndex = i
			break
		}
	}

	if currentIndex == -1 {
		return model.MigrationMethod{}, errors.New("миграция с таким ID не найдена")
	}

	for i := currentIndex + 1; i < len(r.migrationMethods); i++ {
		if r.migrationMethods[i].IsPublished() {
			return r.migrationMethods[i], nil
		}
	}

	return model.MigrationMethod{}, errors.New("это последняя опубликованная миграция")
}

func (r *InMemoryMigrationMethodRepository) FindDraft(ctx context.Context) (model.MigrationMethod, error) {
	for _, m := range r.migrationMethods {
		if m.IsDraft() {
			return m, nil
		}
	}

	return model.MigrationMethod{}, errors.New("черновиков не найдено")
}


func (r *InMemoryMigrationMethodRepository) FindPublishedByTime(ctx context.Context, ltime, rtime float64) ([]model.MigrationMethod, error) {
	var filtered []model.MigrationMethod

	for _, m := range r.migrationMethods {
		if m.IsPublished() && m.TimeInGb >= ltime && m.TimeInGb <= rtime {
			filtered = append(filtered, m)
		}
	}

	if len(filtered) <= 0 {
		return []model.MigrationMethod{}, errors.New("отфильтрованных видео не найдено")
	}

	return filtered, nil
}

func (r *InMemoryMigrationMethodRepository) FindPublished(ctx context.Context) ([]model.MigrationMethod, error) {
	var published []model.MigrationMethod

	for _, m := range r.migrationMethods {
		if m.IsPublished() {
			published = append(published, m)
		}
	}

	if len(published) <= 0 {
		return []model.MigrationMethod{}, errors.New("опубликованных видео не найдено")
	}

	return published, nil
}
