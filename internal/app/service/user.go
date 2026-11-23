package service

import (
	"context"
	"crypto/sha256"
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
)

type Token struct {
	AccessToken  string
	ExpiresIn    int64
}

type UserRepository interface {
	CreateUser(ctx context.Context, user *ds.User) error
	FindUserByEmail(ctx context.Context, email string) (ds.User, error)
	FindUserByID(ctx context.Context, id int64) (ds.User, error)
}

type TokenStore interface {
	IsAccessTokenBlacklisted(ctx context.Context, tokenHash string) (bool, error)
	RevokeAccessToken(ctx context.Context, accessHash string, accessTTL time.Duration) error
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

func (s *UserService) Authenticate(ctx context.Context, email, password string) (Token, error) {
	if email == "" || password == "" {
		return Token{}, ErrInvalidCredentials
	}
	user, err := s.repo.FindUserByEmail(ctx, strings.TrimSpace(email))
	if errors.Is(err, ds.ErrUserNotFound) {
		return Token{}, ErrInvalidCredentials
	}
	if err != nil {
		return Token{}, err
	}
	if user.Password != password {
		return Token{}, ErrInvalidCredentials
	}
	accessToken, _, err := s.tokens.Issue(user.ID)
	if err != nil {
		return Token{}, err
	}
	return Token{AccessToken: accessToken, ExpiresIn: int64(auth.TokenTTL / time.Second)}, nil
}

func accessTokenHash(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func (s *UserService) Logout(ctx context.Context, accessToken string) error {
	_, expiresAt, err := s.tokens.Validate(accessToken)
	if err != nil {
		return err
	}
	return s.store.RevokeAccessToken(ctx, accessTokenHash(accessToken), time.Until(expiresAt))
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
