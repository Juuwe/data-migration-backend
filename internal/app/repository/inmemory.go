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
			Title:       "Онлайн-миграция",
			Description: "Перенос данных в реальном времени с минимальным или нулевым временем простоя системы...",
			VideoKey:    "online.mp4",
			ImageKey:    "online.png",
			Status:      model.StatusPublished,
			TimeInGb:    0.18,
			Reliability: 0.9980,
			Likes:       []int{1, 2},
		},
		{
			ID:          2,
			Title:       "Офлайн-миграция",
			Description: "Разовый пакетный перенос больших объемов данных во время планового технологического окна...",
			VideoKey:    "offline.mp4",
			ImageKey:    "offline.png",
			Status:      model.StatusPublished,
			TimeInGb:    0.12,
			Reliability: 0.9990,
			Likes:       []int{3},
		},
		{
			ID:          3,
			Title:       "Репликация данных",
			Description: "Организация постоянной синхронизации данных между исходной и целевой инфраструктурой...",
			VideoKey:    "replication.mp4",
			ImageKey:    "replication.png",
			Status:      model.StatusDraft,
			TimeInGb:    0.08,
			Reliability: 0.9995,
			Likes:       []int{1},
		},
		{
			ID:          4,
			Title:       "Гибридная миграция",
			Description: "Поэтапный перенос инфраструктуры, сочетающий офлайн-загрузку базового массива данных...",
			VideoKey:    "hybrid.mp4",
			ImageKey:    "hybrid.png",
			Status:      model.StatusPublished,
			TimeInGb:    0.15,
			Reliability: 0.9970,
			Likes:       []int{},
		},
		{
			ID:          5,
			Title:       "ETL-миграция",
			Description: "Перенос данных с их параллельным изменением: заменой формата, изменением схемы базы данных...",
			VideoKey:    "elt.mp4",
			ImageKey:    "etl.png",
			Status:      model.StatusPublished,
			TimeInGb:    0.25,
			Reliability: 0.9950,
			Likes:       []int{2},
		},
		{
			ID:          6,
			Title:       "Физическая миграция",
			Description: "Перенос критически больших массивов данных с использованием физических защищенных накопителей...",
			VideoKey:    "physical.mp4",
			ImageKey:    "appliance.png",
			Status:      model.StatusPublished,
			TimeInGb:    0.05,
			Reliability: 0.9999,
			Likes:       []int{3},
		},
		{
			ID:          7,
			Title:       "Аудит и валидация",
			Description: "Комплексное сопровождение процесса миграции: от разработки стратегии до тестовой верификации...",
			VideoKey:    "audit.mp4",
			ImageKey:    "audit.png",
			Status:      model.StatusPublished,
			TimeInGb:    0.10,
			Reliability: 0.9900,
			Likes:       []int{},
		},
		{
			ID:          8,
			Title:       "Новый метод миграции",
			Description: "Черновик описания...",
			VideoKey:    "",
			ImageKey:    "",
			Status:      model.StatusDraft,
			TimeInGb:    0,
			Reliability: 0,
			Likes:       []int{},
		},
		{
			ID:          9,
			Title:       "Архивная миграция",
			Description: "Карточка логически удаленной услуги...",
			VideoKey:    "",
			ImageKey:    "",
			Status:      model.StatusDeleted,
			TimeInGb:    0.50,
			Reliability: 0.9500,
			Likes:       []int{},
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
