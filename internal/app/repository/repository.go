package repository

import (
	"context"
	"github.com/Juuwe/data-migration-backend/internal/app/model"
)

type MigrationMethodRepository interface {
	FindByID(ctx context.Context, ID int) (model.MigrationMethod, error)
	FindNextPublishedAfterID(ctx context.Context, ID int) (model.MigrationMethod, error)
	FindDraft(ctx context.Context) (model.MigrationMethod, error)
	FindPublishedByTime(ctx context.Context, ltime, rtime float64) ([]model.MigrationMethod, error)
	FindPublished(ctx context.Context) ([]model.MigrationMethod, error)
}
