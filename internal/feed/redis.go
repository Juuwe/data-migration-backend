package feed

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	keyPrefix      = "data-migration-backend:feed:"
	startKeyPrefix = "data-migration-backend:feed-start:"
	feedTTL        = 24 * time.Hour
)

type Store struct {
	client *redis.Client
}

func New(ctx context.Context, address, password string) (*Store, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         address,
		Password:     password,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	})
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("connect feed store to Redis: %w", err)
	}
	return &Store{client: client}, nil
}

func (s *Store) Close() error {
	return s.client.Close()
}

func (s *Store) GetIDs(ctx context.Context, sessionID string) ([]int64, error) {
	key := keyPrefix + sessionID
	values, err := s.client.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("read feed from Redis: %w", err)
	}
	if len(values) > 0 {
		_ = s.client.Expire(ctx, key, feedTTL).Err()
	}

	ids := make([]int64, 0, len(values))
	for _, value := range values {
		id, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse feed item %q: %w", value, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (s *Store) SetIDs(ctx context.Context, sessionID string, ids []int64) error {
	key := keyPrefix + sessionID

	pipe := s.client.TxPipeline()
	pipe.Del(ctx, key)
	if len(ids) > 0 {
		values := make([]any, len(ids))
		for i, id := range ids {
			values[i] = id
		}
		pipe.RPush(ctx, key, values...)
		pipe.Expire(ctx, key, feedTTL)
	}
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("write feed to Redis: %w", err)
	}
	return nil
}

// GetStart возвращает номер первой карточки текущего набора в сквозной
// нумерации сессии. 0 означает, что набор ещё не создавался.
func (s *Store) GetStart(ctx context.Context, sessionID string) (int, error) {
	key := startKeyPrefix + sessionID
	value, err := s.client.Get(ctx, key).Int()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read feed start from Redis: %w", err)
	}
	_ = s.client.Expire(ctx, key, feedTTL).Err()
	return value, nil
}

// SetStart сохраняет номер первой карточки набора в сквозной нумерации сессии.
func (s *Store) SetStart(ctx context.Context, sessionID string, start int) error {
	if err := s.client.Set(ctx, startKeyPrefix+sessionID, start, feedTTL).Err(); err != nil {
		return fmt.Errorf("write feed start to Redis: %w", err)
	}
	return nil
}
