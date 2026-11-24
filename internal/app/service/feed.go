package service

import (
	"context"
	"errors"
)

var ErrFeedEmpty = errors.New("feed is empty")

const DefaultFeedSize = 100

type FeedRepository interface {
	FindRandomPublishedIDs(ctx context.Context, limit int) ([]int64, error)
}

type FeedStore interface {
	GetIDs(ctx context.Context, sessionID string) ([]int64, error)
	SetIDs(ctx context.Context, sessionID string, ids []int64) error
	GetStart(ctx context.Context, sessionID string) (int, error)
	SetStart(ctx context.Context, sessionID string, start int) error
}

type FeedService struct {
	repo  FeedRepository
	store FeedStore
	size  int
}

type FeedCursor struct {
	MethodID int64
	// Position — сквозной номер карточки в сессии: 1–100, 101–200 и так далее.
	Position int
	// Size — сколько карточек сформировано в сессии на текущий момент (100, 200, ...).
	Size int
}

func NewFeedService(repo FeedRepository, store FeedStore, size int) *FeedService {
	if size <= 0 {
		size = DefaultFeedSize
	}
	return &FeedService{repo: repo, store: store, size: size}
}

func (s *FeedService) First(ctx context.Context, sessionID string) (FeedCursor, error) {
	ids, start, err := s.getOrCreate(ctx, sessionID)
	if err != nil {
		return FeedCursor{}, err
	}
	return cursorAt(ids, 0, start)
}

func (s *FeedService) Next(ctx context.Context, sessionID string, currentID int64) (FeedCursor, error) {
	ids, err := s.store.GetIDs(ctx, sessionID)
	if err != nil {
		return FeedCursor{}, err
	}
	start, err := s.store.GetStart(ctx, sessionID)
	if err != nil {
		return FeedCursor{}, err
	}
	if start <= 0 {
		start = 1
	}

	for position, id := range ids {
		if id == currentID && position+1 < len(ids) {
			return cursorAt(ids, position+1, start)
		}
	}

	// Набор закончился или карточка не из него: формируем новые 100 карточек,
	// нумерация в сессии продолжается.
	nextStart := start + len(ids)
	if nextStart < 1 {
		nextStart = 1
	}
	newIDs, newStart, err := s.create(ctx, sessionID, nextStart)
	if err != nil {
		return FeedCursor{}, err
	}
	return cursorAt(newIDs, 0, newStart)
}

func (s *FeedService) getOrCreate(ctx context.Context, sessionID string) ([]int64, int, error) {
	ids, err := s.store.GetIDs(ctx, sessionID)
	if err != nil {
		return nil, 0, err
	}
	start, err := s.store.GetStart(ctx, sessionID)
	if err != nil {
		return nil, 0, err
	}
	if start <= 0 {
		start = 1
	}
	if len(ids) > 0 {
		return ids, start, nil
	}
	return s.create(ctx, sessionID, start)
}

func (s *FeedService) create(ctx context.Context, sessionID string, start int) ([]int64, int, error) {
	if start < 1 {
		start = 1
	}
	ids, err := s.repo.FindRandomPublishedIDs(ctx, s.size)
	if err != nil {
		return nil, 0, err
	}
	if len(ids) == 0 {
		return nil, 0, ErrFeedEmpty
	}
	if err := s.store.SetIDs(ctx, sessionID, ids); err != nil {
		return nil, 0, err
	}
	if err := s.store.SetStart(ctx, sessionID, start); err != nil {
		return nil, 0, err
	}
	return ids, start, nil
}

func cursorAt(ids []int64, position int, start int) (FeedCursor, error) {
	if len(ids) == 0 {
		return FeedCursor{}, ErrFeedEmpty
	}
	if position < 0 || position >= len(ids) {
		position = 0
	}
	if start <= 0 {
		start = 1
	}
	return FeedCursor{
		MethodID: ids[position],
		Position: start + position,
		Size:     start + len(ids) - 1,
	}, nil
}
