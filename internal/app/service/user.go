package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/Juuwe/data-migration-backend/internal/auth"
	"github.com/Juuwe/data-migration-backend/internal/ds"
)

var (
	ErrInvalidEmail        = errors.New("некорректный email")
	ErrInvalidPassword     = errors.New("пароль должен содержать от 8 до 72 байт")
	ErrInvalidCredentials  = errors.New("неверный email или пароль")
	ErrInvalidRefreshToken = errors.New("недействительный refresh-токен")
)

const RefreshTokenTTL = 30 * 24 * time.Hour

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

type UserRepository interface {
	CreateUser(ctx context.Context, user *ds.User) error
	FindUserByEmail(ctx context.Context, email string) (ds.User, error)
	FindUserByID(ctx context.Context, id int64) (ds.User, error)
}

type TokenStore interface {
	SaveRefreshToken(ctx context.Context, userID int64, tokenHash string, ttl time.Duration) error
	RotateRefreshToken(ctx context.Context, oldHash, newHash string, ttl time.Duration) (int64, error)
	IsAccessTokenBlacklisted(ctx context.Context, tokenHash string) (bool, error)
	RevokeTokens(ctx context.Context, userID int64, accessHash, refreshHash string, accessTTL time.Duration) error
}

type UserService struct {
	repo   UserRepository
	tokens *auth.Manager
	store  TokenStore
}

func NewUserService(repo UserRepository, tokens *auth.Manager, store TokenStore) *UserService {
	return &UserService{repo: repo, tokens: tokens, store: store}
}

func (s *UserService) Register(ctx context.Context, email, password string) (ds.User, error) {
	email = strings.TrimSpace(email)
	address, err := mail.ParseAddress(email)
	if email == "" || len(email) > 255 || err != nil || address.Address != email {
		return ds.User{}, ErrInvalidEmail
	}
	if len(password) < 8 || len(password) > 72 {
		return ds.User{}, ErrInvalidPassword
	}
	user := ds.User{Email: email, Password: password}
	if err := s.repo.CreateUser(ctx, &user); err != nil {
		return ds.User{}, err
	}
	return user, nil
}

func (s *UserService) Authenticate(ctx context.Context, email, password string) (TokenPair, error) {
	if email == "" || password == "" {
		return TokenPair{}, ErrInvalidCredentials
	}
	user, err := s.repo.FindUserByEmail(ctx, strings.TrimSpace(email))
	if errors.Is(err, ds.ErrUserNotFound) {
		return TokenPair{}, ErrInvalidCredentials
	}
	if err != nil {
		return TokenPair{}, err
	}
	if user.Password != password {
		return TokenPair{}, ErrInvalidCredentials
	}
	accessToken, _, err := s.tokens.Issue(user.ID)
	if err != nil {
		return TokenPair{}, err
	}
	refreshToken, tokenHash, err := newRefreshToken()
	if err != nil {
		return TokenPair{}, err
	}
	if err := s.store.SaveRefreshToken(ctx, user.ID, tokenHash, RefreshTokenTTL); err != nil {
		return TokenPair{}, err
	}
	return TokenPair{AccessToken: accessToken, RefreshToken: refreshToken, ExpiresIn: int64(auth.TokenTTL / time.Second)}, nil
}

func (s *UserService) Refresh(ctx context.Context, token string) (TokenPair, error) {
	oldHash, err := refreshTokenHash(token)
	if err != nil {
		return TokenPair{}, ErrInvalidRefreshToken
	}
	newToken, newHash, err := newRefreshToken()
	if err != nil {
		return TokenPair{}, err
	}
	userID, err := s.store.RotateRefreshToken(ctx, oldHash, newHash, RefreshTokenTTL)
	if err != nil {
		if errors.Is(err, auth.ErrRefreshTokenNotFound) {
			return TokenPair{}, ErrInvalidRefreshToken
		}
		return TokenPair{}, err
	}
	accessToken, _, err := s.tokens.Issue(userID)
	if err != nil {
		return TokenPair{}, err
	}
	return TokenPair{AccessToken: accessToken, RefreshToken: newToken, ExpiresIn: int64(auth.TokenTTL / time.Second)}, nil
}

func newRefreshToken() (string, string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", err
	}
	token := base64.RawURLEncoding.EncodeToString(bytes)
	hash := sha256.Sum256(bytes)
	return token, hex.EncodeToString(hash[:]), nil
}

func refreshTokenHash(token string) (string, error) {
	if len(token) != 43 {
		return "", ErrInvalidRefreshToken
	}
	bytes, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(bytes) != 32 || base64.RawURLEncoding.EncodeToString(bytes) != token {
		return "", ErrInvalidRefreshToken
	}
	hash := sha256.Sum256(bytes)
	return hex.EncodeToString(hash[:]), nil
}

func accessTokenHash(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func (s *UserService) Logout(ctx context.Context, accessToken, refreshToken string) error {
	userID, expiresAt, err := s.tokens.Validate(accessToken)
	if err != nil {
		return err
	}
	refreshHash, err := refreshTokenHash(refreshToken)
	if err != nil {
		return ErrInvalidRefreshToken
	}
	if err := s.store.RevokeTokens(ctx, userID, accessTokenHash(accessToken), refreshHash, time.Until(expiresAt)); err != nil {
		if errors.Is(err, auth.ErrRefreshTokenNotFound) {
			return ErrInvalidRefreshToken
		}
		return err
	}
	return nil
}

func (s *UserService) UserIDFromToken(ctx context.Context, token string) (int64, error) {
	id, err := s.tokens.UserID(token)
	if err != nil {
		return 0, err
	}
	blacklisted, err := s.store.IsAccessTokenBlacklisted(ctx, accessTokenHash(token))
	if err != nil {
		return 0, err
	}
	if blacklisted {
		return 0, auth.ErrInvalidToken
	}
	if _, err := s.repo.FindUserByID(ctx, id); err != nil {
		if errors.Is(err, ds.ErrUserNotFound) {
			return 0, auth.ErrInvalidToken
		}
		return 0, err
	}
	return id, nil
}
