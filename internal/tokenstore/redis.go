package tokenstore

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Juuwe/data-migration-backend/internal/auth"
	"github.com/redis/go-redis/v9"
)

const keyPrefix = "data-migration-backend:refresh:"

type Store struct {
	client *redis.Client
}

func New(ctx context.Context, address, password string) (*Store, error) {
	client := redis.NewClient(&redis.Options{
		Addr: address, Password: password,
		DialTimeout: 5 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second,
	})
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("connect to Redis: %w", err)
	}
	return &Store{client: client}, nil
}

func (s *Store) Close() error {
	return s.client.Close()
}

func (s *Store) SaveRefreshToken(ctx context.Context, userID int64, tokenHash string, ttl time.Duration) error {
	stored, err := s.client.SetNX(ctx, keyPrefix+tokenHash, strconv.FormatInt(userID, 10), ttl).Result()
	if err != nil {
		return fmt.Errorf("save refresh token: %w", err)
	}
	if !stored {
		return errors.New("refresh token collision")
	}
	return nil
}

func (s *Store) RotateRefreshToken(ctx context.Context, oldHash, newHash string, ttl time.Duration) (int64, error) {
	oldKey, newKey := keyPrefix+oldHash, keyPrefix+newHash
	for {
		var userID string
		err := s.client.Watch(ctx, func(tx *redis.Tx) error {
			var err error
			userID, err = tx.Get(ctx, oldKey).Result()
			if errors.Is(err, redis.Nil) {
				return auth.ErrRefreshTokenNotFound
			}
			if err != nil {
				return err
			}
			exists, err := tx.Exists(ctx, newKey).Result()
			if err != nil {
				return err
			}
			if exists != 0 {
				return errors.New("refresh token collision")
			}
			_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.Set(ctx, newKey, userID, ttl)
				pipe.Del(ctx, oldKey)
				return nil
			})
			return err
		}, oldKey, newKey)
		if errors.Is(err, redis.TxFailedErr) {
			continue
		}
		if errors.Is(err, auth.ErrRefreshTokenNotFound) {
			return 0, err
		}
		if err != nil {
			return 0, fmt.Errorf("rotate refresh token: %w", err)
		}
		id, err := strconv.ParseInt(userID, 10, 64)
		if err != nil || id <= 0 {
			return 0, errors.New("invalid user ID in refresh token store")
		}
		return id, nil
	}
}
