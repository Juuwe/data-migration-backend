package service

import (
	"context"
	"errors"
	"log"

	"github.com/Juuwe/data-migration-backend/internal/app/model"
)

type MigrationMethodRepository interface {
	FindByID(ctx context.Context, ID int) (model.MigrationMethod, error)
	FindNextPublishedAfterID(ctx context.Context, ID int) (model.MigrationMethod, error)
	FindDraft(ctx context.Context) (model.MigrationMethod, error)
	FindPublishedByTime(ctx context.Context, ltime, rtime float64) ([]model.MigrationMethod, error)
	FindPublished(ctx context.Context) ([]model.MigrationMethod, error)
}

type MigrationMethodObjectStorage interface {
	GetURL(ctx context.Context, objectKey string) (string, error)
}


type MigrationMethodService struct {
	repo    MigrationMethodRepository
	storage MigrationMethodObjectStorage
}

func NewMigrationMethodService(repo MigrationMethodRepository, storage MigrationMethodObjectStorage) *MigrationMethodService {
	return &MigrationMethodService{
		repo:    repo,
		storage: storage,
	}
}

type MigrationMethodView struct {
	model.MigrationMethod
	VideoURL   string
	ImageURL   string
	LikesCount int
}

func (s *MigrationMethodService) buildView(ctx context.Context, m model.MigrationMethod) MigrationMethodView {
	videoURL, err := s.storage.GetURL(ctx, m.VideoKey)
	if err != nil {
		log.Printf("[ERROR] Failed to get video URL for key '%s': %v", m.VideoKey, err)
	}

	imageURL, err := s.storage.GetURL(ctx, m.ImageKey)
	if err != nil {
		log.Printf("[ERROR] Failed to get image URL for key '%s': %v", m.ImageKey, err)
	}

	return MigrationMethodView{
		MigrationMethod: m,
		VideoURL:        videoURL,
		ImageURL:        imageURL,
		LikesCount:      len(m.Likes),
	}
}

func (s *MigrationMethodService) GetDraft(ctx context.Context) (MigrationMethodView, error) {
	draft, err := s.repo.FindDraft(ctx)
	if err != nil {
		return MigrationMethodView{}, err
	}
	return s.buildView(ctx, draft), nil
}

func (s *MigrationMethodService) GetPublished(ctx context.Context) ([]MigrationMethodView, error) {
	methods, err := s.repo.FindPublished(ctx)
	if err != nil {
		return nil, err
	}

	views := make([]MigrationMethodView, len(methods))
	for i, m := range methods {
		views[i] = s.buildView(ctx, m)
	}
	return views, nil
}

func (s *MigrationMethodService) GetNextPublishedAfterID(ctx context.Context, id int) (MigrationMethodView, error) {
	m, err := s.repo.FindNextPublishedAfterID(ctx, id)
	if err != nil {
		return MigrationMethodView{}, err
	}
	return s.buildView(ctx, m), nil
}

func (s *MigrationMethodService) GetByID(ctx context.Context, id int) (MigrationMethodView, error) {
	m, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return MigrationMethodView{}, err
	}

	if m.IsDeleted() {
		return MigrationMethodView{}, errors.New("метод удален")
	}

	return s.buildView(ctx, m), nil
}

func (s *MigrationMethodService) GetPublishedByTime(ctx context.Context, minTime, maxTime float64) ([]MigrationMethodView, error) {
	methods, err := s.repo.FindPublishedByTime(ctx, minTime, maxTime)
	if err != nil {
		return []MigrationMethodView{}, err
	}

	views := make([]MigrationMethodView, len(methods))
	for i, m := range methods {
		views[i] = s.buildView(ctx, m)
	}
	return views, nil
}
