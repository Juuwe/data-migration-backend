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

const accessBlacklistPrefix = "data-migration-backend:access-blacklist:"

func (s *Store) IsAccessTokenBlacklisted(ctx context.Context, tokenHash string) (bool, error) {
	count, err := s.client.Exists(ctx, accessBlacklistPrefix+tokenHash).Result()
	if err != nil {
		return false, fmt.Errorf("check access token blacklist: %w", err)
	}
	return count > 0, nil
}

func (s *Store) RevokeTokens(ctx context.Context, userID int64, accessHash, refreshHash string, accessTTL time.Duration) error {
	ttl := (accessTTL + time.Millisecond - 1) / time.Millisecond
	if ttl < 1 {
		ttl = 1
	}
	refreshKey, blacklistKey := keyPrefix+refreshHash, accessBlacklistPrefix+accessHash
	for {
		err := s.client.Watch(ctx, func(tx *redis.Tx) error {
			storedUserID, err := tx.Get(ctx, refreshKey).Result()
			if errors.Is(err, redis.Nil) {
				return auth.ErrRefreshTokenNotFound
			}
			if err != nil {
				return err
			}
			if storedUserID != strconv.FormatInt(userID, 10) {
				return auth.ErrRefreshTokenNotFound
			}
			_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.Del(ctx, refreshKey)
				pipe.Set(ctx, blacklistKey, "1", ttl*time.Millisecond)
				return nil
			})
			return err
		}, refreshKey)
		if errors.Is(err, redis.TxFailedErr) {
			continue
		}
		if errors.Is(err, auth.ErrRefreshTokenNotFound) {
			return err
		}
		if err != nil {
			return fmt.Errorf("revoke tokens: %w", err)
		}
		return nil
	}
}
