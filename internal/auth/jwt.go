package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const TokenTTL = time.Hour
const issuer = "data-migration-backend"

var ErrInvalidToken = errors.New("invalid token")
var ErrRefreshTokenNotFound = errors.New("refresh token not found")

type Manager struct {
	secret []byte
}

func NewManager(secret string) (*Manager, error) {
	if len(secret) < 32 {
		return nil, errors.New("JWT_SECRET must contain at least 32 bytes")
	}
	return &Manager{secret: []byte(secret)}, nil
}

func (m *Manager) Issue(userID int64) (string, time.Time, error) {
	if userID <= 0 {
		return "", time.Time{}, fmt.Errorf("invalid user ID: %d", userID)
	}
	now := time.Now().UTC().Truncate(time.Second)
	expiresAt := now.Add(TokenTTL)
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", time.Time{}, err
	}
	claims := jwt.RegisteredClaims{
		Issuer:    issuer,
		Subject:   strconv.FormatInt(userID, 10),
		ID:        base64.RawURLEncoding.EncodeToString(nonce),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	return token, expiresAt, err
}

func (m *Manager) UserID(tokenString string) (int64, error) {
	id, _, err := m.Validate(tokenString)
	return id, err
}

func (m *Manager) Validate(tokenString string) (int64, time.Time, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(*jwt.Token) (any, error) {
		return m.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(issuer), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if err != nil || !token.Valid {
		return 0, time.Time{}, ErrInvalidToken
	}
	id, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || id <= 0 || claims.ExpiresAt == nil {
		return 0, time.Time{}, ErrInvalidToken
	}
	return id, claims.ExpiresAt.Time, nil
}

type userIDKey struct{}

func WithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, userIDKey{}, userID)
}

func UserID(ctx context.Context) int64 {
	id, _ := ctx.Value(userIDKey{}).(int64)
	return id
}
