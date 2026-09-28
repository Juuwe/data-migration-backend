package service

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/Juuwe/data-migration-backend/internal/app/serializer"
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
	ErrInvalidDescription = errors.New("описание не может быть пустым")
	ErrInvalidTimeInGb    = errors.New("время на Гб должно быть больше нуля")
	ErrInvalidReliability = errors.New("коэффициент надежности должен быть от 0 до 1")
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

func (s *MigrationMethodService) serializeMethod(m ds.MigrationMethod, likesCount int, currentUserID int64) serializer.MigrationMethod {
	if s.media == nil {
		return serializer.NewMethod(m, likesCount, currentUserID, DefaultImageURL, DefaultVideoURL)
	}
	return serializer.NewMethod(m, likesCount, currentUserID,
		s.media.publicURL(m.ImageKey, DefaultImageURL),
		s.media.publicURL(m.VideoKey, DefaultVideoURL))
}

func (s *MigrationMethodService) serializeSingleMethod(ctx context.Context, m *ds.MigrationMethod, currentUserID int64) (serializer.MigrationMethod, error) {
	likesCount, err := s.repo.CountMethodLikesByID(ctx, m.ID)
	if err != nil {
		log.Printf("[WARN] Failed to get likes count for method %d: %v", m.ID, err)
	}

	return s.serializeMethod(*m, likesCount, currentUserID), nil
}

func (s *MigrationMethodService) serializeMethodList(ctx context.Context, methods []ds.MigrationMethod, currentUserID int64) ([]serializer.MigrationMethod, error) {
	if len(methods) == 0 {
		return make([]serializer.MigrationMethod, 0), nil
	}

	methodIDs := make([]int64, len(methods))
	for i := range methods {
		methodIDs[i] = methods[i].ID
	}

	likesMap, err := s.repo.CountLikesByMethodIDs(ctx, methodIDs)
	if err != nil {
		log.Printf("Failed to fetch likes count batch: %v", err)
	}

	result := make([]serializer.MigrationMethod, len(methods))
	for i := range methods {
		id := methods[i].ID
		result[i] = s.serializeMethod(methods[i], likesMap[id], currentUserID)
	}

	return result, nil
}

func (s *MigrationMethodService) GetDraft(ctx context.Context, creatorID int64) (serializer.MigrationMethod, bool, error) {
	draft, err := s.repo.FindDraft(ctx, creatorID)
	if err != nil {
		if errors.Is(err, ds.ErrMigrationMethodNotFound) {
			return serializer.MigrationMethod{}, false, nil
		}
		return serializer.MigrationMethod{}, false, err
	}
	return s.serializeMethod(draft, 0, creatorID), true, nil
}

func (s *MigrationMethodService) GetPublished(ctx context.Context, currentUserID int64) ([]serializer.MigrationMethod, error) {
	methods, err := s.repo.FindPublished(ctx)
	if err != nil {
		return nil, err
	}

	return s.serializeMethodList(ctx, methods, currentUserID)
}

func (s *MigrationMethodService) GetNextPublishedAfterID(ctx context.Context, id, currentUserID int64) (serializer.MigrationMethod, error) {
	m, err := s.repo.FindNextPublishedAfterID(ctx, id)
	if err != nil {
		if !errors.Is(err, ds.ErrMigrationMethodNotFound) {
			return serializer.MigrationMethod{}, err
		}

		published, findErr := s.repo.FindPublished(ctx)
		if findErr != nil {
			return serializer.MigrationMethod{}, findErr
		}
		if len(published) == 0 {
			return serializer.MigrationMethod{}, ds.ErrMigrationMethodNotFound
		}
		m = published[0]
	}

	return s.serializeSingleMethod(ctx, &m, currentUserID)
}

func (s *MigrationMethodService) GetByID(ctx context.Context, id, currentUserID int64) (serializer.MigrationMethod, error) {
	m, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return serializer.MigrationMethod{}, err
	}

	if !m.IsPublished() {
		return serializer.MigrationMethod{}, ds.ErrMigrationMethodNotFound
	}

	return s.serializeSingleMethod(ctx, &m, currentUserID)
}

func (s *MigrationMethodService) GetPublishedByTime(ctx context.Context, minTime, maxTime float64, currentUserID int64) ([]serializer.MigrationMethod, error) {
	methods, err := s.repo.FindPublishedByTime(ctx, minTime, maxTime)
	if err != nil {
		return []serializer.MigrationMethod{}, err
	}

	return s.serializeMethodList(ctx, methods, currentUserID)
}

func (s *MigrationMethodService) CreateDraftMethod(ctx context.Context, title string, creatorID int64, imageInput, videoInput MediaInput) (serializer.MigrationMethod, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return serializer.MigrationMethod{}, ErrDraftTitleRequired
	}
	if len([]rune(title)) > 255 {
		return serializer.MigrationMethod{}, ErrDraftTitleTooLong
	}

	_, exists, err := s.GetDraft(ctx, creatorID)
	if err != nil {
		return serializer.MigrationMethod{}, err
	}
	if exists {
		return serializer.MigrationMethod{}, ErrDraftAlreadyExists
	}
	if s.media == nil {
		return serializer.MigrationMethod{}, errors.New("хранилище медиа не настроено")
	}

	image, err := s.media.prepareImage(imageInput)
	if err != nil {
		return serializer.MigrationMethod{}, err
	}
	video, err := s.media.prepareVideo(videoInput)
	if err != nil {
		return serializer.MigrationMethod{}, err
	}
	if err := s.media.upload(ctx, image); err != nil {
		return serializer.MigrationMethod{}, errors.Join(err, s.media.cleanup(ctx, image.key))
	}
	if err := s.media.upload(ctx, video); err != nil {
		return serializer.MigrationMethod{}, errors.Join(err, s.media.cleanup(ctx, image.key, video.key))
	}

	m := ds.MigrationMethod{
		Title:     title,
		Status:    ds.StatusDraft,
		CreatorID: creatorID,
		ImageKey:  image.key,
		VideoKey:  video.key,
	}
	if err := s.repo.Create(ctx, &m); err != nil {
		return serializer.MigrationMethod{}, errors.Join(err, s.media.cleanup(ctx, image.key, video.key))
	}
	return s.serializeMethod(m, 0, creatorID), nil
}

func (s *MigrationMethodService) PublishDraft(ctx context.Context, creatorID int64, desc string, timeInGb, reliability float64) (serializer.MigrationMethod, error) {
	desc = strings.TrimSpace(desc)
	if desc == "" {
		return serializer.MigrationMethod{}, ErrInvalidDescription
	}
	if timeInGb <= 0 {
		return serializer.MigrationMethod{}, ErrInvalidTimeInGb
	}
	if reliability < 0 || reliability > 1 {
		return serializer.MigrationMethod{}, ErrInvalidReliability
	}

	draft, err := s.repo.FindDraft(ctx, creatorID)
	if err != nil {
		if errors.Is(err, ds.ErrMigrationMethodNotFound) {
			return serializer.MigrationMethod{}, ds.ErrMigrationMethodNotFound
		}
		return serializer.MigrationMethod{}, err
	}

	draft.Description = &desc
	draft.TimeInGb = &timeInGb
	draft.Reliability = &reliability
	draft.Status = ds.StatusPublished
	publishedAt := time.Now()
	draft.PublishedAt = &publishedAt

	if err := s.repo.Update(ctx, &draft); err != nil {
		return serializer.MigrationMethod{}, err
	}
	return s.serializeSingleMethod(ctx, &draft, creatorID)
}

func (s *MigrationMethodService) DeleteMethod(ctx context.Context, id, creatorID int64) error {
	if id <= 0 {
		return errors.New("некорректный ID")
	}
	return s.repo.SoftDeleteSQL(ctx, id, creatorID)
}

var ErrInvalidLike = errors.New("значение like должно быть 0 или 1")

func (s *MigrationMethodService) SetLike(ctx context.Context, methodID, userID int64, like int) (serializer.MigrationMethodLike, error) {
	if methodID <= 0 || userID <= 0 {
		return serializer.MigrationMethodLike{}, errors.New("некорректный ID")
	}
	if like != 0 && like != 1 {
		return serializer.MigrationMethodLike{}, ErrInvalidLike
	}

	method, err := s.repo.FindByID(ctx, methodID)
	if err != nil {
		return serializer.MigrationMethodLike{}, err
	}
	if !method.IsPublished() {
		return serializer.MigrationMethodLike{}, ds.ErrMigrationMethodNotFound
	}

	if err := s.repo.SetLike(ctx, methodID, userID, like); err != nil {
		return serializer.MigrationMethodLike{}, err
	}
	likesCount, err := s.repo.CountMethodLikesByID(ctx, methodID)
	if err != nil {
		return serializer.MigrationMethodLike{}, err
	}
	return serializer.NewLike(methodID, like == 1, likesCount), nil
}
