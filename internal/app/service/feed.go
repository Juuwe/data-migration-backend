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
}

type FeedService struct {
	repo  FeedRepository
	store FeedStore
	size  int
}

type FeedCursor struct {
	MethodID int64
	Position int
	Size     int
}

func NewFeedService(repo FeedRepository, store FeedStore, size int) *FeedService {
	if size <= 0 {
		size = DefaultFeedSize
	}
	return &FeedService{repo: repo, store: store, size: size}
}

func (s *FeedService) First(ctx context.Context, sessionID string) (FeedCursor, error) {
	ids, err := s.getOrCreate(ctx, sessionID)
	if err != nil {
		return FeedCursor{}, err
	}
	return cursorAt(ids, 0)
}

func (s *FeedService) Next(ctx context.Context, sessionID string, currentID int64) (FeedCursor, error) {
	ids, err := s.store.GetIDs(ctx, sessionID)
	if err != nil {
		return FeedCursor{}, err
	}

	for position, id := range ids {
		if id == currentID && position+1 < len(ids) {
			return cursorAt(ids, position+1)
		}
	}

	ids, err = s.create(ctx, sessionID)
	if err != nil {
		return FeedCursor{}, err
	}
	return cursorAt(ids, 0)
}

func (s *FeedService) getOrCreate(ctx context.Context, sessionID string) ([]int64, error) {
	ids, err := s.store.GetIDs(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if len(ids) > 0 {
		return ids, nil
	}
	return s.create(ctx, sessionID)
}

func (s *FeedService) create(ctx context.Context, sessionID string) ([]int64, error) {
	ids, err := s.repo.FindRandomPublishedIDs(ctx, s.size)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, ErrFeedEmpty
	}
	if err := s.store.SetIDs(ctx, sessionID, ids); err != nil {
		return nil, err
	}
	return ids, nil
}

func cursorAt(ids []int64, position int) (FeedCursor, error) {
	if len(ids) == 0 {
		return FeedCursor{}, ErrFeedEmpty
	}
	if position < 0 || position >= len(ids) {
		position = 0
	}
	return FeedCursor{
		MethodID: ids[position],
		Position: position + 1,
		Size:     len(ids),
	}, nil
}
