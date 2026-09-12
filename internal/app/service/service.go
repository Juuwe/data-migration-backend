package service

import (
	"context"
	"errors"
	"log"
	"time"
	"strings"

	"github.com/Juuwe/data-migration-backend/internal/ds"
)

const (
	DefaultImageURL = "/static/images/default.jpg"
	DefaultVideoURL = "/static/videos/default.mp4"
)

type MigrationMethodRepository interface {
	FindByID(ctx context.Context, ID int) (ds.MigrationMethod, error)
	FindNextPublishedAfterID(ctx context.Context, ID int) (ds.MigrationMethod, error)
	FindDraft(ctx context.Context, creatorID int64) (ds.MigrationMethod, error)
	FindPublishedByTime(ctx context.Context, ltime, rtime float64) ([]ds.MigrationMethod, error)
	FindPublished(ctx context.Context) ([]ds.MigrationMethod, error)

	CountMethodLikesByID(ctx context.Context, methodID int64) (int, error)
	CountLikesByMethodIDs(ctx context.Context, methodIDs []int64) (map[int64]int, error)

	Create(ctx context.Context, method *ds.MigrationMethod) error
	Update(ctx context.Context, method *ds.MigrationMethod) error
	SoftDeleteSQL(ctx context.Context, id int64) error
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
	ds.MigrationMethod
	VideoURL   string
	ImageURL   string
	LikesCount int
}

func (s *MigrationMethodService) buildView(ctx context.Context, m *ds.MigrationMethod, likesCount int) MigrationMethodView {
	videoURL, err := s.storage.GetURL(ctx, m.VideoKey)
	if err != nil {
		log.Printf("[ERROR] Failed to get video URL for key '%s': %v", m.VideoKey, err)
	}

	imageURL, err := s.storage.GetURL(ctx, m.ImageKey)
	if err != nil {
		log.Printf("[ERROR] Failed to get image URL for key '%s': %v", m.ImageKey, err)
	}

	return MigrationMethodView{
		MigrationMethod: *m,
		VideoURL:        videoURL,
		ImageURL:        imageURL,
		LikesCount:      likesCount,
	}
}

func (s *MigrationMethodService) buildSingleView(ctx context.Context, m *ds.MigrationMethod) (MigrationMethodView, error) {
	likesCount, err := s.repo.CountMethodLikesByID(ctx, m.ID)
	if err != nil {
		log.Printf("[WARN] Failed to get likes count for method %d: %v", m.ID, err)
	}

	return s.buildView(ctx, m, likesCount), nil
}

func (s *MigrationMethodService) buildViewList(ctx context.Context, methods []ds.MigrationMethod) ([]MigrationMethodView, error) {
	if len(methods) == 0 {
		return make([]MigrationMethodView, 0), nil
	}

	methodIDs := make([]int64, len(methods))
	for i := range methods {
		methodIDs[i] = methods[i].ID
	}

	likesMap, err := s.repo.CountLikesByMethodIDs(ctx, methodIDs)
	if err != nil {
		log.Printf("Failed to fetch likes count batch: %v", err)
	}

	views := make([]MigrationMethodView, len(methods))
	for i := range methods {
		id := methods[i].ID
		views[i] = s.buildView(ctx, &methods[i], likesMap[id])
	}

	return views, nil
}

func (s *MigrationMethodService) GetDraft(ctx context.Context, creatorID int64) (MigrationMethodView, bool, error) {
	draft, err := s.repo.FindDraft(ctx, creatorID)
	if err != nil {
		return MigrationMethodView{}, false, nil
	}
	return s.buildView(ctx, &draft, 0), true, nil
}

func (s *MigrationMethodService) GetPublished(ctx context.Context) ([]MigrationMethodView, error) {
	methods, err := s.repo.FindPublished(ctx)
	if err != nil {
		return nil, err
	}

	return s.buildViewList(ctx, methods)
}

func (s *MigrationMethodService) GetNextPublishedAfterID(ctx context.Context, id int) (MigrationMethodView, error) {
	m, err := s.repo.FindNextPublishedAfterID(ctx, id)
	if err != nil {
		return MigrationMethodView{}, err
	}

	return s.buildSingleView(ctx, &m)
}

func (s *MigrationMethodService) GetByID(ctx context.Context, id int) (MigrationMethodView, error) {
	m, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return MigrationMethodView{}, err
	}

	if m.IsDeleted() {
		return MigrationMethodView{}, errors.New("метод удален")
	}


	return s.buildSingleView(ctx, &m)
}

func (s *MigrationMethodService) GetPublishedByTime(ctx context.Context, minTime, maxTime float64) ([]MigrationMethodView, error) {
	methods, err := s.repo.FindPublishedByTime(ctx, minTime, maxTime)
	if err != nil {
		return []MigrationMethodView{}, err
	}

	return s.buildViewList(ctx, methods)
}

func (s *MigrationMethodService) CreateDraftMethod(ctx context.Context, title string, creatorID int64) error {
	if strings.TrimSpace(title) == "" {
		return errors.New("название не может быть пустым")
	}

	_, exists, _ := s.GetDraft(ctx, creatorID)
	if exists {
		return errors.New("черновик уже существует")
	}

	m := ds.MigrationMethod{
		Title:     title,
		Status: ds.StatusDraft,
		CreatorID: creatorID,
		FormedAt:  time.Now(),
	}
	return s.repo.Create(ctx, &m)
}

func (s *MigrationMethodService) PublishDraft(ctx context.Context, creatorID int64, desc string, timeInGb, reliability float64) error {
	draft, err := s.repo.FindDraft(ctx, creatorID)
	if err != nil {
		return errors.New("черновик не найден")
	}

	draft.Description = desc
	draft.TimeInGb = timeInGb
	draft.Reliability = reliability
	draft.Status = ds.StatusPublished
	draft.FormedAt = time.Now()

	return s.repo.Update(ctx, &draft)
}

func (s *MigrationMethodService) DeleteMethod(ctx context.Context, id int64) error {
	return s.repo.SoftDeleteSQL(ctx, id)
}
