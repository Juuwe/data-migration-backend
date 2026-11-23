package tokenstore

import (
	"context"
	"fmt"
	"time"
)

const accessBlacklistPrefix = "data-migration-backend:access-blacklist:"

func (s *Store) IsAccessTokenBlacklisted(ctx context.Context, tokenHash string) (bool, error) {
	count, err := s.client.Exists(ctx, accessBlacklistPrefix+tokenHash).Result()
	if err != nil {
		return false, fmt.Errorf("check access token blacklist: %w", err)
	}
	return count > 0, nil
}

func (s *Store) RevokeAccessToken(ctx context.Context, accessHash string, accessTTL time.Duration) error {
	ttl := (accessTTL + time.Millisecond - 1) / time.Millisecond
	if ttl < 1 {
		ttl = 1
	}
	if err := s.client.Set(ctx, accessBlacklistPrefix+accessHash, "1", ttl*time.Millisecond).Err(); err != nil {
		return fmt.Errorf("revoke access token: %w", err)
	}
	return nil
}
