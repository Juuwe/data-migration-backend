package service

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/Juuwe/data-migration-backend/internal/ds"
)

const (
	DefaultImageURL = "/static/images/default.svg"
	DefaultVideoURL = "/static/videos/default.mp4"
)

var (
	ErrDraftTitleRequired = errors.New("название не может быть пустым")
	ErrDraftTitleTooLong  = errors.New("название не может быть длиннее 255 символов")
	ErrDraftAlreadyExists = errors.New("черновик уже существует")
)

type MigrationMethodRepository interface {
	FindByID(ctx context.Context, id int64) (ds.MigrationMethod, error)
	FindNextPublishedAfterID(ctx context.Context, id int64) (ds.MigrationMethod, error)
	FindDraft(ctx context.Context, creatorID int64) (ds.MigrationMethod, error)
	FindPublishedByTime(ctx context.Context, ltime, rtime float64) ([]ds.MigrationMethod, error)
	FindPublished(ctx context.Context) ([]ds.MigrationMethod, error)

	CountMethodLikesByID(ctx context.Context, methodID int64) (int, error)
	CountLikesByMethodIDs(ctx context.Context, methodIDs []int64) (map[int64]int, error)

	Create(ctx context.Context, method *ds.MigrationMethod) error
	Update(ctx context.Context, method *ds.MigrationMethod) error
	SoftDeleteSQL(ctx context.Context, id, creatorID int64) error
	SetLike(ctx context.Context, methodID, userID int64, like int) error
}

type MigrationMethodService struct {
	repo  MigrationMethodRepository
	media *MediaService
}

func NewMigrationMethodService(repo MigrationMethodRepository, media ...*MediaService) *MigrationMethodService {
	var mediaService *MediaService
	if len(media) > 0 {
		mediaService = media[0]
	}
	return &MigrationMethodService{
		repo:  repo,
		media: mediaService,
	}
}

type MigrationMethodView struct {
	ds.MigrationMethod
	ImageURL             string  `json:"image_url"`
	VideoURL             string  `json:"video_url"`
	CreatedByCurrentUser bool    `json:"created_by_current_user"`
	LikesCount           int     `json:"likes_count"`
	TimeInGb             float64 `json:"time_in_gb"`
	Reliability          float64 `json:"reliability"`
}

func (s *MigrationMethodService) buildView(m ds.MigrationMethod, likesCount int, currentUserID int64) MigrationMethodView {
	view := MigrationMethodView{
		MigrationMethod:      m,
		CreatedByCurrentUser: currentUserID > 0 && m.CreatorID == currentUserID,
		LikesCount:           likesCount,
	}
	if s.media == nil {
		view.ImageURL, view.VideoURL = DefaultImageURL, DefaultVideoURL
	} else {
		view.ImageURL = s.media.publicURL(m.ImageKey, DefaultImageURL)
		view.VideoURL = s.media.publicURL(m.VideoKey, DefaultVideoURL)
	}
	if m.TimeInGb != nil {
		view.TimeInGb = *m.TimeInGb
	}
	if m.Reliability != nil {
		view.Reliability = *m.Reliability
	}
	return view
}

func (s *MigrationMethodService) buildSingleView(ctx context.Context, m *ds.MigrationMethod, currentUserID int64) (MigrationMethodView, error) {
	likesCount, err := s.repo.CountMethodLikesByID(ctx, m.ID)
	if err != nil {
		log.Printf("[WARN] Failed to get likes count for method %d: %v", m.ID, err)
	}

	return s.buildView(*m, likesCount, currentUserID), nil
}

func (s *MigrationMethodService) buildViewList(ctx context.Context, methods []ds.MigrationMethod, currentUserID int64) ([]MigrationMethodView, error) {
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
		views[i] = s.buildView(methods[i], likesMap[id], currentUserID)
	}

	return views, nil
}

func (s *MigrationMethodService) GetDraft(ctx context.Context, creatorID int64) (MigrationMethodView, bool, error) {
	draft, err := s.repo.FindDraft(ctx, creatorID)
	if err != nil {
		if errors.Is(err, ds.ErrMigrationMethodNotFound) {
			return MigrationMethodView{}, false, nil
		}
		return MigrationMethodView{}, false, err
	}
	return s.buildView(draft, 0, creatorID), true, nil
}

func (s *MigrationMethodService) GetPublished(ctx context.Context, currentUserID int64) ([]MigrationMethodView, error) {
	methods, err := s.repo.FindPublished(ctx)
	if err != nil {
		return nil, err
	}

	return s.buildViewList(ctx, methods, currentUserID)
}

func (s *MigrationMethodService) GetNextPublishedAfterID(ctx context.Context, id, currentUserID int64) (MigrationMethodView, error) {
	m, err := s.repo.FindNextPublishedAfterID(ctx, id)
	if err != nil {
		if !errors.Is(err, ds.ErrMigrationMethodNotFound) {
			return MigrationMethodView{}, err
		}

		published, findErr := s.repo.FindPublished(ctx)
		if findErr != nil {
			return MigrationMethodView{}, findErr
		}
		if len(published) == 0 {
			return MigrationMethodView{}, ds.ErrMigrationMethodNotFound
		}
		m = published[0]
	}

	return s.buildSingleView(ctx, &m, currentUserID)
}

func (s *MigrationMethodService) GetByID(ctx context.Context, id, currentUserID int64) (MigrationMethodView, error) {
	m, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return MigrationMethodView{}, err
	}

	if !m.IsPublished() {
		return MigrationMethodView{}, ds.ErrMigrationMethodNotFound
	}

	return s.buildSingleView(ctx, &m, currentUserID)
}

func (s *MigrationMethodService) GetPublishedByTime(ctx context.Context, minTime, maxTime float64, currentUserID int64) ([]MigrationMethodView, error) {
	methods, err := s.repo.FindPublishedByTime(ctx, minTime, maxTime)
	if err != nil {
		return []MigrationMethodView{}, err
	}

	return s.buildViewList(ctx, methods, currentUserID)
}

func (s *MigrationMethodService) CreateDraftMethod(ctx context.Context, title string, creatorID int64, imageInput, videoInput MediaInput) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return ErrDraftTitleRequired
	}
	if len([]rune(title)) > 255 {
		return ErrDraftTitleTooLong
	}

	_, exists, err := s.GetDraft(ctx, creatorID)
	if err != nil {
		return err
	}
	if exists {
		return ErrDraftAlreadyExists
	}
	if s.media == nil {
		return errors.New("хранилище медиа не настроено")
	}

	image, err := s.media.prepareImage(imageInput)
	if err != nil {
		return err
	}
	video, err := s.media.prepareVideo(videoInput)
	if err != nil {
		return err
	}
	if err := s.media.upload(ctx, image); err != nil {
		return errors.Join(err, s.media.cleanup(ctx, image.key))
	}
	if err := s.media.upload(ctx, video); err != nil {
		return errors.Join(err, s.media.cleanup(ctx, image.key, video.key))
	}

	m := ds.MigrationMethod{
		Title:     title,
		Status:    ds.StatusDraft,
		CreatorID: creatorID,
		ImageKey:  image.key,
		VideoKey:  video.key,
	}
	if err := s.repo.Create(ctx, &m); err != nil {
		return errors.Join(err, s.media.cleanup(ctx, image.key, video.key))
	}
	return nil
}

func (s *MigrationMethodService) PublishDraft(ctx context.Context, creatorID int64, desc string, timeInGb, reliability float64) error {
	desc = strings.TrimSpace(desc)
	if desc == "" {
		return errors.New("описание не может быть пустым")
	}
	if timeInGb <= 0 {
		return errors.New("время на Гб должно быть больше нуля")
	}
	if reliability < 0 || reliability > 1 {
		return errors.New("коэффициент надежности должен быть от 0 до 1")
	}

	draft, err := s.repo.FindDraft(ctx, creatorID)
	if err != nil {
		if errors.Is(err, ds.ErrMigrationMethodNotFound) {
			return errors.New("черновик не найден")
		}
		return err
	}

	draft.Description = &desc
	draft.TimeInGb = &timeInGb
	draft.Reliability = &reliability
	draft.Status = ds.StatusPublished
	publishedAt := time.Now()
	draft.PublishedAt = &publishedAt

	return s.repo.Update(ctx, &draft)
}

func (s *MigrationMethodService) DeleteMethod(ctx context.Context, id, creatorID int64) error {
	if id <= 0 {
		return errors.New("некорректный ID")
	}
	return s.repo.SoftDeleteSQL(ctx, id, creatorID)
}

var ErrInvalidLike = errors.New("значение like должно быть 0 или 1")

func (s *MigrationMethodService) SetLike(ctx context.Context, methodID, userID int64, like int) error {
	if methodID <= 0 || userID <= 0 {
		return errors.New("некорректный ID")
	}
	if like != 0 && like != 1 {
		return ErrInvalidLike
	}

	method, err := s.repo.FindByID(ctx, methodID)
	if err != nil {
		return err
	}
	if !method.IsPublished() {
		return ds.ErrMigrationMethodNotFound
	}

	return s.repo.SetLike(ctx, methodID, userID, like)
}
